# Recuperacion del estado


## Persistencia del estado 

Para persistir el estado en los diferentes workers, se utiliza el metodo siguiente:

```go

for WorkIsLeftToDo() {
    msg := RecoverFromMiddleware()

    DoWork(msg)

    PrepareState()

    PublishResults()

    CommitState()

    AckMiddleware(msg)
}

```

En el pseudocodigo anterior se muestra como se persiste el estado a medida que un worker procesa porciones del trabajo total:

    1. Se toma una porcion del trabajo
    2. Se realiza el trabajo pertinente
    3. Se prepara un archivo nuevo de estado temporal
    4. Se publican los resultados al middleware (esperando confirmacion de que el middleware recibio esos resultados)
    5. Se reemplaza el archivo de estado por el archivo temporal previamente escrito
    6. Se le avisa al middleware que esa porcion del trabajo se encuentra realizada.

Si bien el metodo general es el expuesto, cada tipo de worker tiene una particularidad. Estas particularidades se exponen a continuacion.

##### Particularidades: DemuxFilter

La unica particularidad que tiene un worker de este tipo es la siguiente:

1. Cuando el worker termina de procesar todos los datos del cliente, guarda en su estado que termino con esta etapa y que va a encargarse de enviar los datos de la suma de las tarifas y el total de vuelos procesados a los AvgFilters.
2. Cuando envia estos datos, el DemuxFilter guarda en su estado que termino de enviar estos datos.

##### Particularidades: DistanceFilter

La particularidad del distance filter esta en la forma que utiliza para procesar las coordenadas de los aeropuertos que envia un cliente. Esto lo realiza de la siguiente manera:

```go

PrepareState()
for CoordinatesLeft() {
    coords := GetCoordsFromMiddleware()
    AppendCoordinatesToCoordinateFile(coords)
    AckMiddleware()
}
CommitState()
```

La operacion de append a un archivo se considera atomica, por lo que es una manera de persistir el estado en este punto. Ademas, en caso de tener por alguna razon, un par de coordenadas repetidas en el archivo, el resultado que se obtendra sera equivalente al resultado que se obtiene al no tener repetidos (ya que las coordendas son las mismas).

El hecho de commitear el estado al finalizar con este procesamiento se debe a que de esta forma, se puede identificar si un worker de este tipo termino de procesar las coordenadas de un cliente o no.

##### Particularidades: FastestFilter

El filtro por tiempo de trayecto utiliza la siguiente etrategia:

1. Mientras recibo vuelos, los mas rapidos de cada trayecto se guardan en un archivo dedicado

La estrategia es la siguiente:

```go
PrepareState()
for FlightsLeft() {
    msg := RecoverFromMiddleware()

    DoWork(msg)

    UpdateFlightFile(msg)

    AckMiddleware(msg)
}
CommitState()
```

Esta estrategia es practicamente igual a la estrategia utilizada por el distanceFilter para manejar las coordenadas de los aeropuertos.
La razon por la que utiliza esta estrategia es por que es facil determinar el estado en el que se encontraba el filtro al momento de reiniciar su ejecucion luego de un crash. Si tiene un estado, entonces hay que enviar resultados, si no, hay que esperar mas vuelos.

2. Mientras envia los resultados, utiliza la siguiente estrategia:

```go

for ResultsLeftToSend() {
    result := GetResult()

    PrepareState()

    PublishResults(result) //Se asegura que los resultados hayan sido recibidos correctamente por Rabbit

    CommitState()

}

```

##### Particularidades: AvgFilter

El filtro por promedio tiene la necesidad de pesistir todas las tarifas de cada trayecto antes de poder procesar los datos, por lo que la estrategia que se utiliza durante el envio de vuelos es la siguiente:

```go
for FlightsLeft(){
    msg := RecoverFromMiddleware()

    if NewRoute() {
        AddRouteToState()
    }

    AppendFlightFareToFile(msg)

    PrepareState()
    CommitState()

    AckMiddleware(msg)
}
```

Este filtro guarda dos cosas en el estado durante esta fase:

1. Los trayectos que se conocen
2. El total de vuelos para cada uno de los trayectos 

Estos dos datos se necesitan al momento de recuperar el estado del filtro por la siguiente razon:

1. Si el filtro crashea despues de haber escrito un nuevo vuelo, pero antes de hacer ACK al middleware, se va a tener que 
truncar el archivo para ese trayecto particular. 
2. Si el filtro crashea, se tienen que reabrir todos los archivos donde se escriben las tarifas para cada uno de los diferentes trayectos.


Durante la fase de envio de resultados el AvgFilter persiste su estado de la siguiente manera:

```go

for ResultsLeftToSend() {
    result := RecoverResult()

    PrepareState()

    PublishResult() //Se asegura que el mensaje fue recibido correctamente por el middleware

    CommitState()
}
```

Esto lo hace para conocer hasta que punto envio resultados en caso de haber crasheado el filtro


## Recuperacion del estado

El metodo de recuperacion del estado funciona de forma ligeramente diferente para cada uno de los tipos de workers,
por eso, se detalla a continuacion la recuperacion del estado para los diferentes tipos de workers


##### Recuperacion del estado: DemuxFilter

El DemuxFilter recupera su estado de la siguiente manera:

```go

if HasState() {
    switch RunningState{
        case RecievingData:
            AddToRecievers()
        case ResultsNotYetSent:
            RestartFilter()
        case Finished:
            EndExecution()
    }
}

for MoreClients() {
    client := GetClient()

    if ClientInRecievers(client) {
        RestartFilterWith(client)
    }
    StartNewFilter(client)
}
```

Basicamente, lo que hace el demuxfilter es lo siguiente:

1. Si todavia se encontraba recibiendo datos de vuelos desde el cliente, entonces se lo agrega a una estructura para que, cuando llegue un nuevo mensaje de ese cliente, el filtro pueda resumir su ejecucion.
2. Si Todavia no habia finalizado de enviar los resultados (suma total de tarifas y total de vuelos recibidos), entonces resume la ejecucion inmediatamente.
3. Si ya habia terminado de enviar los datos, pero el filtro crasheo antes de poder eliminar su estado, entonces simplemente finaliza su ejecucion.



##### Recuperacion del estado: DistanceFilter

El DistanceFilter recupera su estado de la siguiente manera:

```go

if HasState() {
    PrepareFilterToWaitForFlights()
}

```

Este tipo de worker puede recuperar su estado de esta manera, por que el archivo de estado lo crea una vez que terminado de procesar la totalidad de las coordenadas de aeropuertos.

El hecho de que al tener un estado ya se sepa que el worker esta esperando vuelos de un cliente se debe al hecho mencionado anteriormente que las coordenadas las persiste un archivo diferente al del estado.
En caso de que el worker no haya terminado de procesar las coordenadas, este archivo sigue disponible para continuar siendo escrito (con lineas agregadas al final del mismo.)

##### Recuperacion del estado: AvgFilter y FastestFilter

El AvgFilter y el FastestFilter recuperan su estado de la siguiente manera:

```go
if HasState() {
    switch RunningState {
        case RecievingData:
            AddToRecievers()
        case SendingData:
            ResumeSendingResults()
    }
}

for MoreClients() {
    client := GetClient()

    if ClientInRecievers(client) {
        RestartFilterWith(client)
    }
    StartNewFilter(client)
}
```

Este funcionamiento es similar al del demuxFilter, con la excepcion de que no tiene un finished state. En caso de reiniciarse el filtro luego de haber completado el envio de los resultados pero sin haber eliminado del archivo de estado, simplemente va a terminar su ejecucion

