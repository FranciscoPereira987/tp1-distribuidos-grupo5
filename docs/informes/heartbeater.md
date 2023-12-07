## Heartbeater

### Diagrama de actividades

![ActividadesHeartbeater](../../img/ActividadesHeartbeat.png)

El diagrama muestra de forma general el funcionamiento logico del heartbeater, tanto como ejecuta la eleccion de lider[^1] como su comportamiento segun sea o bien un lider o bien un cliente.

- En caso de comportarse como cliente, el heartbeater se encarga de controlar el estado del lider y responder a sus controles.
- En caso de comportase como lider, el heartbeater se encarga de responder los controles de los peers y controlar el estado de todos los contenedores del sistema, reinstanciandolos de ser necesario.

[^1]: La eleccion de lider se realiza utilizando el algoritmo de *invitation*, se puede encontrar una explicacion y pseudocodigo del mismo en [el siguiente informe](leaderElection.md)