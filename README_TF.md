# Tolerancia a fallos

Este documento, tiene los mismos contenidos que se pueden encontrar por separado en los siguientes archivos:

- [Modelo c4 del sistema](docs/informes/c4.md)
- [Actividades input](docs/informes/inputBoundary.md)
- [Actividades demuxFilter](docs/informes/demuxFilter.md)
- [Actividades distanceFilter](docs/informes/distanceFilter.md)
- [Actividades fastestFilter](docs/informes/fastestFilter.md)
- [Actividades avgFilter](docs/informes/avgFilter.md)
- [Actividades output](docs/informes/outputBoundary.md)
- [Recuperacion de estado](docs/informes/stateRecovery.md)
- [Filtrado de mensajes duplicados](docs/informes/duplicateFilter.md)
- [Control de estado de workers](docs/informes/heartbeater.md)
- [Eleccion de lider](docs/informes/leaderElection.md)

## Diagramas C4

### Contexto

![Diagrama de contexto para _Flight Optimizer_](img/C4-Capa1.png)

El cliente utiliza la consola (IO) para enviar los datos de las coordenadas de
los aeropuertos y los vuelos, y recibir los resultados de las queries del
sistema. 


### Contenedores

![Diagrama de contenedores para _Flight Optimizer_](img/C4-Capa2.png)

#### Cliente

El cliente recibe resultados desde la consola, que muestra los muestra mediante
*stdout* y ademas los guarda en 4 archivos *.csv* correspondientes a cada una
de las queries:

1. first.csv
2. second.csv
3. third.csv
4. fourth.csvs

#### Flight optimizer

El sistema se despliega en siete contenedores:

1. Parser $\longrightarrow$ recibe los archivos de coordenadas y vuelos del
   cliente, los parsea y envía a los workers que corresponda (coordenadas al
   filtro por distancias y vuelos as demux).

2. Agregador de Resultados $\longrightarrow$ Recibe los resultados obtenidos y
   los envía a la consola en formato CSV (en este caso por red).

3. Demux $\longrightarrow$ Toma los datos de vuelos y los envía a los demás
   workers, filtra los vuelos con 3 o más escalas y calcula la media general de
   precios para enviarla a **Promedio y Máximo**.

4. Filtro distancia $\longrightarrow$ Se encarga de filtrar aquellos vuelos
   cuya distancia total sea mayor a 4 veces la distancia directa entre los
   aeropuertos.

5. Vuelos mas rapidos $\longrightarrow$ Se encarga de filtrar los dos vuelos
   mas rapidos para cada trayecto entre todos aquellos vuelos que tengan 3 o
   mas escalas.

6. Promedio y maximo $\longrightarrow$ Se encarga de agregar, para cada
   trayecto, los precios de todos los vuelos y posteriormente filtrar aquellos
   precios que sean menores que el promedio general de precios y calcular, para
   cada trayecto el promedio y maximo segun lo filtrado.

7. Heartbeater $\longrightarrow$ Se encarga de, mediante el envio de mensajes, asegurarse que el resto de los contenedores del 
   sistema se encuentran disponibles y en caso de que no lo esten, reinstanciarlos mediantes llamadas al daemon de *Docker*

### Componentes

#### Interfaz

![capa3Interfaz](img/C4-Capa3-Boundaries.png)

Para realizar su tarea, los datos que recibe y parsea el _input boundary_, los
distribuye a travez de diferentes exchanges de *Rabbitmq*. Envía las
coordenadas al exchange de coordenadas (tipo _fanout_) para que las replique en
todos los filtros por distancia y envía los datos de vuelos a los _demuxers_
para que los procesen y envíen a los demás workers.

Por su parte el _output boundary_ toma los resultados de una cola y los envía
al cliente en formato CSV.

- [Diagrama de código para el input boundary](inputBoundary.md)
- [Diagrama de código para el output boundary](outputBoundary.md)

#### Demux

![capa3Demux](img/C4-Capa3-Demux.png)

Cada _demux_ procesa una porción de los datos, y los envía a los demás
_workers_, o a la cola de resultados. Cuando no hay más datos que procesar
envía la suma y cantidad de tarifas a los filtros por promedio.

- [Diagrama de código para el demux](demuxFilter.md)

#### Filtro por distancias

![capa3Distancias](img/C4-Capa3-Distance.png)

El filtro por distancia consta de dos componentes. Que se ejecutan en el mismo
proceso. En un principio los datos que llegan a la cola de coordenadas se
guardan para su posterior utilizacion (son necesarios para calcular las
distancias directas entre aeropuertos).  
Posteriormente, los datos de vuelos son filtrados segun la relacion entre la
distancia total recorrida en el vuelo y la distancia directa entre aeropuertos.  
Aquellos vuelos que cumplan con la condicion de tener una distancia total
recorrida mayor a 4 veces la distancia directa entre aeropuertos son enviados a
la cola de resultados.

- [Diagrama de código para el filtro distancias](distanceFilter.md)

#### Filtro por Duración de Vuelo

![capa3Duracion](img/C4-Capa3-Fastest.png)

El filtro de vuelos mas rapidos toma datos de una cola en donde existen
unicamente vuelos que tengan 3 o mas escalas.  
Utilizando estos vuelos, para cada trayecto, calcula los dos vuelos de menor
duracion.  
Finalmente, envia los resultados obtenidos por la cola de resultados.

- [Diagrama de código para el filtro por duración](avgFilter.md)

#### Filtro por Media General de Precios

![capa3Promedio](img/C4-Capa3-Average.png)

El filtro por promedio toma los vuelos desde la cola correspondiente y agrupa
los precios de los vuelos segun el trayecto de los mismos.  
Una vez calculado el promedio general de precios, el filtro lo recibe, filtra
aquellos vuelos de cada trayecto que tengan un precio menor a este y con los
precios restantes, calcula el maximo y el promedio de precios de cada trayecto.  
Los resultados los envia a la cola de resultados.

- [Diagrama de código para el filtro por promedio](avgFilter.md)

#### Heartbeater

![capa3Heartbeater](img/C4-Capa3-Heartbeater.png)

El proceso de heartbeater se encarga, por un lado, de asegurarse que todos los workers se encuentren *vivos* y por otro de reiniciar a aquellos workers que se encuentren *muertos*. 
Para llevar a cabo esta tarea, el heartbeater envia *Heartbeats* a los diferentes workers, recibiendo un *Ok* de parte de aquellos workers que se encuentren *vivos*.
Al mismo, como coexisten multiples heartbeaters en el sistema, los mismos deben decidir cual de ellos es el *leader*. El lider cumple el rol antes descripto y los *members* se aseguran de que el lider continue vivo. 
En caso de que los *members* consideren que el *leader* no se encuentra disponible, se eligira un nuevo lider.
Los request que el heartbeater realiza para reinstanciar a otros contenedores, los realiza a traves de la API de docker.

- [Diagrama de código para el hertbeater](heartbeater.md)


## Parser

Un parser que se encarga de tomar los datos enviados por el cliente y
enviarselos a los filtros por distancia si son coordenadas, o a los demux si
son datos de vuelos.

### Diagrama de actividades

Describe el funcionamiento del parser y el mecansimo de _graceful shutdown_.

![actividades](../../img/DiagramaActividadesInput.png)

El parser comienza haciendo setup del middleware, esperando a que todos los
_workers_ esten listos para así empezar a enviar datos, y empieza a escuchar
por conexiones de clientes.

Luego para cada cliente, inicia o reinicia la conexión con el mismo. Si es un
cliente nuevo, este le envia los dos archivos (coordenadas y vuelos), y si se
está reanudando la conexión, el parser le indica en donde se había quedado para
que el cliente continue enviandole datos desde ahí (utilizando seek(2)). Al
terminar le indica al cliente que procesó todos los datos.


1. Si los datos son coordenadas los envia directamente al filtro por distancias.
2. Si son datos de vuelos, los envia siempre al demux.
3. Una vez finalizado el envio de datos, el parser notifica al cliente.

### Diagramas de secuencia

Los diagramas de secuencia muestran la comunicacion de la interfaz en diferentes momentos de la ejecucion del sistema.

#### Alta de workers

![secuenciaAltaWorkers](../../img/SecuenciaAltaWorkers.png)

El diagrama muestra la comunicacion entre los diferentes filtros y el parser,
mientras esta espere a que todos los workers le comuniquen que se encuentran
listos para comenzar a procesar datos (*workerUp*/*Ready*).  
Luego, el parser puede comenzar a atender clientes.

#### Conexion y envio de datos del cliente

![secuenciaDatos](../../img/SecuenciaConexionYEnvioDeDatos.png)

El diagrama muestra la comunicacion entre el cliente, el parser y RabbitMQ a
medida que se envian los datos, tanto de coordenadas como de vuelos (el envio
de coordenadas ocurre previo al envio de vuelos).  
Ademas, se muestran los datos que se envian a cada exchange.

## Demultiplexador de vuelos

### Diagrama de actividades
Describe el funcionamiento del filtro, su interacción con el middleware y el
mecansimo de _graceful shutdown_.

![actividades](../../img/DiagramaActividadesQ1.png)

## Filtro Distancias

El diagrama de actividades muestra la logica general implementada para el filtro de distancias (*cmd/distanceFilter*) mientras que el diagrama de secuencia muestra la comunicacion que realiza el mismo.

### Diagrama de actividades

![actividades](../../img/DiagramaActividadesQ2.png)

El filtro de distancia comienza haciendo setups tanto de middleware como del propio worker. Luego instancia una goroutine en la cual se ejecuta el filtro en si, la goroutine principal se encarga de esperar por *SIGINT/SIGTERM* y a que termine de ejecutarse el filtro.
El filtro espera por datos, en base a si los mismos corresponden a coordenadas o datos de vuelos realiza una operacion u otra.
Si se procesa un vuelo y el mismo corresponde a un vuelo que cumple la condicion del filtro, el vuelo es enviado a la cola de resultados (yieldFlight).
Finalmente, el filtro anuncia que termino su trabajo y la main goroutine se encarga de liberar los recursos.

### Diagrama de secuencia

![secuencia](../../img/SecuenciaFiltroDistancia.png)

El filtro se encarga, en primer lugar de avisar que se encuentra listo para recibir datos.
Posteriormente se encarga de procesar coordenadas hasta que comienzan a llegar datos de vuelos. En este punto el filtro procesa vuelos hasta que se termina el stream de vuelos del cliente. En este punto el filtro anuncia que termino su trabajo.


## Filtro por Duración

### Diagrama de actividades
Describe el funcionamiento del filtro, su interacción con el middleware y el
mecansimo de _graceful shutdown_.

![actividades](../../img/DiagramaActividadesQ3.png)

## Filtro por Media General de Precios

### Diagrama de actividades
Describe el funcionamiento del filtro, su interacción con el middleware y el
mecansimo de _graceful shutdown_.

![actividades](../../img/DiagramaActividadesQ4.png)

## Agregador de Resultados

El agregador se encarga de recibir los resultados obtenidos por cada filtro y
enviarselos al cliente. Ademas de estar pendiente del momento en que termina el
procesamiento de los datos.

### Diagrama de actividades

Describe el funcionamiento del filtro, su interacción con el middleware y el
mecansimo de _graceful shutdown_.

![actividades](../../img/DiagramaActividadesOutput.png)

El agregador espera por resultados directamente y lo hace hasta que hayan
llegado todos los resultados y todos los workers hayan terminado (esto se
realiza mediante avisos de cada uno de los workers de que su ejecucion termino
y la lógica es propia del middleware).

### Diagramas de secuencia

Los diagramas de secuencia muestran la comunicacion del agregador en diferentes
momentos de la ejecucion del sistema.

#### Conexion y envio de datos del cliente

![secuenciaDatos](../../img/SecuenciaConexionYEnvioDeDatos.png)

El diagrama muestra la comunicacion entre el cliente, la interface y RabbitMQ a medida que se envian los datos, tanto de coordenadas como de vuelos (el envio de coordenadas ocurre previo al envio de vuelos).
Ademas, se muestran los datos que se envian a cada exchange.
Cuando el cliente finaliza el envio de sus datos, el parser envia el promedio al filtro por promedios y un mensaje de control a cada uno de los otros exchanges.

#### Secuencia envio de resultados

![resultados](../../img/SecuenciaEnvioDeResultados.png)

El diagrama muestra la comunicacion entre el cliente, el agregador y RabbitMQ
para el envio de los resultados.  
Se muestra como siempre que existan mas resultados por enviar, los mismos son
tomados de una cola de RabbitMQ (`results`) y enviados al cliente.

#### Secuencia fin de datos

![final](../../img/SecuenciaFinTrabajo.png)

El diagrama muestra la forma en la que todos los workers comunican que
terminaron su trabajo. Mostrando como, cuando se cumple la condicion de que
cada uno de los workers anuncian que terminaron su trabajo. El agregador le
comunica al cliente que se termino el procesamiento de sus datos.


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

#### Particularidades: Inputboundary

El input boundary usa el metodo anteriormente mencionado con la salvedad de que el trabajo no lo toma del middleware, si no que el trabajo lo obtiene de la conexion con el cliente. Por ello, tambien se guarda el offset leido hasta el momento.

#### Particularidades: Outputboundary

El outputboundary tambien utiliza el metodo mencionado, con la particularidad de que como no realiza ningun tabajo sobre los datos que obtiene del middleware, su metodo seria el siguiente:

```go
for ResultsLeftToSend() {
    result := GetResultFromMiddleware()

    PrepareState()
    SendResult()
    CommitState()

    AckMiddleware()
}
```

#### Particularidades: DemuxFilter

La unica particularidad que tiene un worker de este tipo es la siguiente:

1. Cuando el worker termina de procesar todos los datos del cliente, guarda en su estado que termino con esta etapa y que va a encargarse de enviar los datos de la suma de las tarifas y el total de vuelos procesados a los AvgFilters.
2. Cuando envia estos datos, el DemuxFilter guarda en su estado que termino de enviar estos datos.

#### Particularidades: DistanceFilter

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

#### Particularidades: FastestFilter

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

#### Particularidades: AvgFilter

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

### Recuperacion del estado con filtros sin afinidad

Tanto el Demuxfilter, como el distanceFilter son un tipo de worker que no tienen afinidad por el trayecto de un vuelo en particular. Si bien cada filtro tiene su propia cola de Rabbit, el worker que envia datos a un grupo de filtros sin afinidad lo hace utilizando un mecanismo de *Round Robin*.
Por esta razon, tanto el input como el demuxFilter, persisten el estado del RoundRobin propio para que, al momento de recuperar su estado y enviar un mensaje duplicado, el mismo no se envie a distintos filtros (esto generaria una duplicacion real de los mensajes y afectaria el resultado de las queries.)

- El demuxFilter realiza *Round Robin* para comunicarse con los distanceFilters
- El input realiza *Round Robin* para comunicarse con los demuxFilters

#### Recuperacion del estado: DemuxFilter

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

#### Recuperacion del estado: DistanceFilter

El DistanceFilter recupera su estado de la siguiente manera:

```go

if HasState() {
    PrepareFilterToWaitForFlights()
}

```

Este tipo de worker puede recuperar su estado de esta manera, por que el archivo de estado lo crea una vez que terminado de procesar la totalidad de las coordenadas de aeropuertos.

El hecho de que al tener un estado ya se sepa que el worker esta esperando vuelos de un cliente se debe al hecho mencionado anteriormente que las coordenadas las persiste un archivo diferente al del estado.
En caso de que el worker no haya terminado de procesar las coordenadas, este archivo sigue disponible para continuar siendo escrito (con lineas agregadas al final del mismo.)

#### Recuperacion del estado: AvgFilter y FastestFilter

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

## Filtrado de mensajes duplicados

Una garantia que RabbitMQ provee que el sistema utiliza a su favor es que RabbitMQ se asegura que los mensajes se reencolan en orden.
Esto permite que el algoritmo utilizado para filtrar duplicados sea el siguiente:

```go

func IsDuplicate(msg) bool {
    return lastPeerSender[msg.SenderId] == msg.Id
}

func FilterDuplicate(msg) bool {
    duplicate := IsDuplicate(msg)
    lastPeerSender[msg.SenderId] = msg.Id
    return duplicate
}

```

Gracias a la garantia de Rabbit mencionada anteriormente, se puede determinar si un mensaje es o no duplicado si el mensaje es el mismo que el procesado anteriormente para ese sender en particular.

El hecho de que se utilize la identidad del sender se debe a que cada filtro genera un Id incremental antes de enviar un mensaje a traves de Rabbit. Por esta razon, si no diferenciamos segun quien envio el mensaje, podriamos filtrar un mensaje que tenga igual Id a otro pero que sin embargo provenga de una fuente diferente al mensaje original.

Es importante aclarar que el resto de los filtros deben persistir el estado del Filtro de duplicados, para que en caso de soportar un crash, no vuelvan a procesar un lote de datos que ya haya sido procesado por filtro y cuyo resultado pudo haber sido incluso persistido por el mismo.

### Caso especial: FastestFilter

El filtro por mas rapidos es un caso especial al momento de filtrar mensajes duplicados. Esto se debe a que este filtro en particular no tiene la necesidad de filtrar los mensajes debido a su propio proposito.
El FastestFilter es un filtro idempotente frente a mensajes duplicados, por que a lo sumo reemplazara el valor de un vuelo mas rapido por su mismo valor. 

## Control de estado de los workers (Heartbeater)


### Diagrama C4 de capa 3

![capa3Heartbeater](img/C4-Capa3-Heartbeater.png)

El proceso de heartbeater se encarga, por un lado, de asegurarse que todos los workers se encuentren *vivos* y por otro de reiniciar a aquellos workers que se encuentren *muertos*. 
Para llevar a cabo esta tarea, el heartbeater envia *Heartbeats* a los diferentes workers, recibiendo un *Ok* de parte de aquellos workers que se encuentren *vivos*.
Al mismo, como coexisten multiples heartbeaters en el sistema, los mismos deben decidir cual de ellos es el *leader*. El lider cumple el rol antes descripto y los *members* se aseguran de que el lider continue vivo. 
En caso de que los *members* consideren que el *leader* no se encuentra disponible, se eligira un nuevo lider.
Los request que el heartbeater realiza para reinstanciar a otros contenedores, los realiza a traves de la API de docker.

### Diagrama de actividades

![ActividadesHeartbeater](img/ActividadesHeartbeat.png)

El diagrama muestra de forma general el funcionamiento logico del heartbeater, tanto como ejecuta la eleccion de lider como su comportamiento segun sea o bien un lider o bien un cliente.

- En caso de comportarse como cliente, el heartbeater se encarga de controlar el estado del lider y responder a sus controles.
- En caso de comportase como lider, el heartbeater se encarga de responder los controles de los peers y controlar el estado de todos los contenedores del sistema, reinstanciandolos de ser necesario.

##### Beater Server

El beater server funciona con 5 rutinas:

1. Una rutina que actua como main
2. Una rutina que se encarga unicamente de escribir en el socket UDP
3. Una rutina que se encarga unicamente de leer del socket UDP
4. Una rutina que se encarga unicamente de comunicarse con el daemon de docker
5. Una rutina que se encarga de iterar por cada uno de los clientes en ciclo
    - Esta rutina envia a traves de un channel el nombre de los clientes que considera muertos para que se revivan
    - Lee de un channel los mensajes del socket
    - Escribe en un channel los pedidos de respuesta a los clientes.

## Algoritmo de invitacion

Para la eleccion de lider en los heartbeaters se utilizo el algoritmo de invitacion. Este algoritmo se elegio debido a que, en caso de crash de un peer, al ser este reinstanciado por el lider del grupo, no se tendra que volver a ejecutar una eleccion y simplemente el peer reinstanciado pasara a formar parte del grupo existente.

### Pseudocodigo

```go
    state := ELECTING
    group := []
    for {
        switch state {
            case ELECTING:
                switch message {
                    case INVITE:
                        if message.groupSize >= len(group){
                            send(ACCEPT to message.ID)
                            state = MEMBER
                            send(CHANGE to group members)
                        }else{
                            send(REJECT to message.ID)
                        }
                    case ACCEPT:
                        addToGroup(message.members)
                    case REJECT:
                        if message.ID == IDPeerLastSend {
                            send(ACCEPT to message.ID)
                            state = MEMBER
                        }else{
                            send(INVITE to message.ID)
                        }
                    case NULL:
                        invite(peer not in group)
                        if group contains all responding peers{
                            state = COORDINATOR
                        }
                }
            case MEMBER:
                switch message {
                    case INVITE:
                        send(REJECT to message.ID)
                    case CHANGE:
                        changeCoordinator(message.ID)
                    case NULL:
                        send(HEARTBEAT to Coordinator)
                        if Coordinator not responding {
                            state = ELECTING
                        }
                }
            case COORDINATOR:
                respond(HEARTBEAT from Peers)
                switch message {
                    case INVITE:
                        if message.groupSize >= len(group){
                            send(ACCEPT to message.ID)
                            state = MEMBER
                            send(CHANGE to group members)
                        }else{
                            send(REJECT to message.ID)
                        }
                    case ACCEPT:
                        addToGroup(message.members)
                    case REJECT:
                        if message.ID == IDPeerLastSend {
                            send(ACCEPT to message.ID)
                            state = MEMBER
                            send(CHANGE to group members)
                        }else{
                            send(INVITE to message.ID)
                        }
                    case HEARTBEAT:
                        send(OK to message.ID)
                    case NULL:
                        send(invite to non-responding peers)
                }
        } 
    }
``` 
El pseudocodigo muestra la logica seguida en cada uno de sus estados posibles. Un nodo puede o bien estar ejecutando una eleccion, siendo miembro de un grupo, o siendo coordinador de un grupo.
Durante una eleccion, un nodo que no tiene mensajes que responder va a intentar invitar a otro nodo a que forme parte del grupo que el coordina (INVITE). 
En caso de aceptar (ACCEPT), el otro nodo le envia a todos los peers que forman parte de su grupo quien es el nuevo coordinador (CHANGE) y comienza a ser coordinado por el nuevo lider del grupo.
En caso de que el primer nodo reciba un (REJECT) existen dos posibilidades:

    1. Que quien envia el reject sea lider de grupo
    2. Que quien envia el reject no sea lider de grupo

En el primer caso, el nodo asume que el grupo del otro peer es mas grande que el propio, por lo que responde con ACCEPT y posteriormente avisando a los peers que forman parte de su propio grupo mediante el envio de CHANGE quien es el nuevo coordinador. En el segundo caso, el nodo debe enviar la invitacion a quien es el lider del otro grupo.

Cuando un nodo actua como member de un grupo, tiene la responsabilidad de asegurarse que el lider se encuentra vivo. Para esto, le envia al lider mensajes de HEARTBEAT, que el lider tiene la responsabilidad de contestar con OK.
Un member, frente a mensajes de INVITE responde con REJECT, junto con el ID de su lider. A su vez, frente a un CHANGE, el member cambia la identidad del lider de su grupo.

Cuando un nodo actua como lider de un grupo, tiene la responsabilidad de contestar con OK a todos los HEARTBEATs que reciba de parte de los miembros de su grupo. Frente a mensajes de INVITE, el lider puede responder con ACCEPT o REJECT (segun el tamaño del grupo de quien lo invita). Tambien 
se tiene que encargar de procesar mensajes ACCEPT o REJECT de peers que previamente no hayan estado disponibles y de enviar invitaciones a estos peers segun corresponda.


###### Referencias

- Petrov, Alex. (2019). Database Internals (First edition). Publisher. O’Reilly Media, Inc.
    - Capitulo 10