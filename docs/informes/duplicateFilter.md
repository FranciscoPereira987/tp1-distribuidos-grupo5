# Filtrado de mensajes duplicados

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
