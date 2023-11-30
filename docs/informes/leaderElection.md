## Algoritmo de invitacion

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