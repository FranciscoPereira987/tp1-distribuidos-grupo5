# TP1: FLight Optimizer (Sistemas Distribuidos, FIUBA, 2do Cuatrimestre 2023)

- [enunciado](docs/Enunciado.pdf)
- [Diagramas C4](docs/informes/c4.md)
- [Instrucciones de uso](docs/informes/instrucciones.md)


### Dependencias (DAG)

Este diagrama describe las dependencias que hay entre distintas etapas del
procesamiento, el espesor de las flechas indica el flujo de los datos a travez
de las mismas (excepto por los datos de coordenadas que no tienen correlación
con los datos de vuelos), y las tuplas indican que información proveen.

- La flecha más tenue es la que solo pasa un valor, que es la media general de
  precios.
- Luego están los resultados cuyo flujo solo depende de la cantidad de
  trayectos:
    + Calcular el promedio y el máximo por trayecto, y
    + Los resultados de los 2 vuelos más rápidos por trayecto.
- A estos les siguen los vuelos filtrados por cantidad de escalas y por
  distancia.
- Y las cuatro mas espesas son los datos de los vuelos sin filtrar.

![fotoDependencias](img/DAG.png)

### Sistema

![fotoSistema](img/DiagramaRobustez.png)

### Despliegue

![fotoDespliegue](img/DiagramaDespliegue.png)

El despliegue del sistema se hacen en 5 tipos de contenedores. 

1. Uno en el cual se ejecuta la interfaz del sistema.
2. Otro contenedor en el cual se ejecuta el filtro de distancia, que es escalable en cantidad de contenedores desplegados.
3. Un tipo de contenedor para el filtro de los vuelos mas rapidos. Que tambien es escalable.
4. Un tipo de contenedor para el filtro por promedio, que tambien es escalable.

Todos estos contenedores dependen del ultimo, en el cual se ejecuta RabbitMQ, que se utiliza para las comunicaciones entre los diferentes grupos de contenedores.

### Diagrama de paquetes

![fotoPaquetes](img/DiagramaPaquetes.png)

El diagrama muestra las dependencias entre los paquetes de codigo implementados para el sistema.

1. distance $\longrightarrow$ Implementa una estructura que se encarga de guardar las coordenadas de cada uno de los aeropuertos y calcular las distancias directas entre cualesquiera dos aeropuertos. 
2. Middleware $\longrightarrow$ Implementa la interfaz utilizada para la comunicacion contra RabbitMQ que utiliza cada uno de los ejecutables del sistema.
3. Protocol $\longrightarrow$ Implementa el protocolo de comunicacion entre el cliente y la interfaz. El protocolo utiliza TLV para el envio de datos.
4. Typing $\longrightarrow$ Implementa el TLV de cada tipo de dato que el cliente envia al servidor (coordenadas y vuelos) y que el servidor envia al cliente (uno por cada tipo de resultado)
5. connection $\longrightarrow$ Implementa una interfaz para que protocol no tenga que ocuparse de manejar toda la logica necesaria para evitar short-reads y short-writes de los sockets.
6. Reader $\longrightarrow$ Implementa funcionalidad de lectura y parseo basico de archivos csv. 
7. utils $\longrightarrow$ Implementa utilidades generales como interfaces basicas, escritura, configuracion y logueo.