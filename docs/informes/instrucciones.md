## Instrucciones de uso

### Configuracion del cliente

Para configurar el cliente es necesario seguir los siguientes pasos:

1. En la carpeta client, agregar los archivos test.csv y airports-codepublic.csv en el subdirectorio data. Estos archivos contienen la informacion de los vuelos y las coordenadas de los aeropuertos respectivamente.

2. Crear una carpeta results en el mismo directorio. En esta carpeta se guardan los resultados de las 4 queries luego de una corrida del cliente.
    - first.csv contiene los resultados de la primer query
    - second.csv contiene los resultados de la segunda query
    - third.csv contiene los resultados de la tercera query
    - fourth.csv contiene los resultados de la cuarta query
    
### Ejecutar el sistema

Para ejecutar el sistema es necesario correr los siguientes comandos:

1. ```bash
        make build-image
    ```
    - Este comando construye las imagenes de docker.
2. ```bash
        WORKERS_QUERY2=N WORKERS_QUERY3=M WORKERS_QUERY4=P make setup 
    ```
    - Este comando construye el archivo de docker compose. Se puede ejecutar sin necesidad de setear las variables. Pero al setear las variables se puede cambiar la cantidad de workers para cada una de las queries.
3. ```bash
        make docker-compose-up
    ```
    - Este comando crea los contenedores y levanta el sistema con la configuracion previamente establecida.
4. ```bash
        make run-client
    ```
    - Este comando ejecuta el cliente