# TP1: FLight Optimizer (Sistemas Distribuidos, FIUBA, 2do Cuatrimestre 2023)

- [enunciado](docs/Enunciado.pdf)
- [Diagramas C4](docs/informes/c4.md)
- [Instrucciones de uso](docs/informes/instrucciones.md)


### Dependencias (DAG)

Este diagrama describe las dependencias que hay entre distintas etapas del
procesamiento, el espesor de las flechas indica el flujo de los datos a travez
de las mismas (excepto por los datos de coordenadas que no tienen correlación
con los datos de vuelos), y las tuplas indican que información proveen.

![fotoDependencias](img/DAG.png)

- La flecha más tenue es la que solo pasa un valor, que es la media general de
  precios.
- Luego están los resultados cuyo flujo solo depende de la cantidad de
  trayectos:
    + Calcular el promedio y el máximo por trayecto, y
    + Los resultados de los 2 vuelos más rápidos por trayecto.
- A estos les siguen los vuelos filtrados por cantidad de escalas y por
  distancia.
- Y las cuatro mas espesas son los datos de los vuelos sin filtrar.

### Sistema

Lo que sigue es el diagrama de robustez. Comenzando desde arriba tenemos al
**cliente**, quien establece dos conexiones, una con el **parser**
(_inputBoundary_), a quien le envía los datos de vuelos y coordenadas de
aeropuertos, y otra con el **agregador de resultados** (_outputBoundary_),
quien le envía los resultados.  
El **parser**, primero espera a que todos los _workers_ esten listos (quienes
le avisan con un mensaje _Ready_ en la cola _status_), toma las coordenadas de
los aeropuertos y las envía a el **filtro por distancia** (_distanceFilter_),
quien las guarda para luego calcular la distancia entre aeropuertos, luego toma
los datos de vuelos los proyecta sobre las columnas que necesita el sistema y
los envía al **demux** (_demuxFilter_).  
Por su parte el **demux** toma los datos de vuelos y los proyecta sobre las
columnas que necesita cada _worker_, filtra los vuelos con 3 o más escalas para
enviarlos al **filtro por duración** (_distanceFilter_) y publicarlos como
resultados, y calcula la media general de precios para enviarsela a el **filtro
por promedio**[^1] (_avgFilter_).  
En el caso de los _workers_ que responden a las consultas 3 y 4 (los filtros
por duración y promedio), estos utilizan un esquema de _sharding_ en el
despliegue con afinidad a los trayectos que nos permite procesar los vuelos
según el trayecto recorrido (origen-destino), sin necesidad de sincronización
entre ellos.

![fotoSistema](img/DiagramaRobustez.png)

[^1]: Es fácil ver que el costo de calcular la media general de precios es
    despreciable comparado con el costo de enviar un valor para que se calcule
    la media en otros nodos.

### Despliegue

![fotoDespliegue](img/DiagramaDespliegue.png)

El despliegue del sistema se realiza con los siguientes tipos de contenedores:

1. RabbitMQ $\longrightarrow$ Es un contenedor de RabbitMQ
2. InputFilter $\longrightarrow$ El proceso de este contenedor se encarga de la comunicacion directa con los clientes
3. Demux $\longrightarrow$ En este contenedor se ejecuta el proceso que deserializa los datos del cliente, se filtran por cantidad de escalas, se calculan las sumas y cantidades de vuelos por ciente y se derivan los datos a los diferentes contenedores a traves de Rabbit.
4. AvgFilter $\longrightarrow$ En este contenedor se ejecuta el proceso para filtrar los vuelos segun su precio con respecto al promedio y realizar la agregacion por trayecto.
5. FastestFilter $\longrightarrow$ Aqui se ejecuta el filtro para los vuelos mas rapidos por trayecto que tengan 3 o mas escalas.
6. DistanceFilter $\longrightarrow$ Aqui se ejecuta el filtro de vuelos cuya distancia total recorrida sea mayor a 4 veces la distancia directa entre el aeropuerto de partida y destino.
7. Heartbeater $\longrightarrow$ En este contenedor se ejecuta el proceso encargado de asegurarse que el resto de los contenedores se encuentren *vivos*
  - Si bien el heartbeater no depende directamente de Rabbit, el proceso depende de el resto de los contenedores del sistema y como el resto de los procesos si dependen de Rabbit, heartbeater tambien depende de rabbit de forma transitoria.


Todos estos contenedores dependen del ultimo, en el cual se ejecuta RabbitMQ, que se utiliza para las comunicaciones entre los diferentes grupos de contenedores.

### Diagrama de paquetes

![fotoPaquetes](img/DiagramaPaquetes.png)

El diagrama muestra las dependencias entre los paquetes de codigo implementados para el sistema.

1. distance $\longrightarrow$ Implementa una estructura que se encarga de guardar las coordenadas de cada uno de los aeropuertos y calcular las distancias directas entre cualesquiera dos aeropuertos. 
2. Middleware $\longrightarrow$ Implementa la interfaz utilizada para la comunicacion contra RabbitMQ que utiliza cada uno de los ejecutables del sistema.
3. Protocol $\longrightarrow$ Implementa el protocolo de comunicacion entre el cliente y los boundaries (input y output).
4. Typing $\longrightarrow$ Implementa el TLV de cada tipo de dato que el cliente envia al servidor (coordenadas y vuelos) y que el servidor envia al cliente (uno por cada tipo de resultado)
5. connection $\longrightarrow$ Implementa una interfaz para que protocol no tenga que ocuparse de manejar toda la logica necesaria para evitar short-reads y short-writes de los sockets.
7. utils $\longrightarrow$ Implementa utilidades generales como interfaces basicas, escritura, configuracion y logueo.
8. invitation $\longrightarrow$ Implementa el algoritmo de [*invitation*](docs/informes/leaderElection.md).
9. beater  $\longrightarrow$ Implementa tanto el servidor como el cliente que se utiliza para asegurarse que los distintos workers se encuentren *vivos*
10. dood $\longrightarrow$ Implementa la interfaz con docker para reinicializar contenedores.
11. duplicates $\longrightarrow$ Implementa el filtro de mensajes duplicados.
12. state $\longrightarrow$ Implementa el estado de los workers, tanto su persistencia como recuperacion.
