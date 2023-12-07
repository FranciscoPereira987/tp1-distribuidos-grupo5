## Instrucciones de uso

### Configuracion del cliente

Para configurar el cliente es necesario seguir los siguientes pasos:

1. En la carpeta client, agregar los archivos test.csv y airports-codepublic.csv en el subdirectorio data. Estos archivos contienen la informacion de los vuelos y las coordenadas de los aeropuertos respectivamente.

2. Crear una carpeta results en el mismo directorio. En esta carpeta se guardan los resultados de las 4 queries luego de una corrida del cliente.
    - first.csv contiene los resultados de la primer query
    - second.csv contiene los resultados de la segunda query
    - third.csv contiene los resultados de la tercera query
    - fourth.csv contiene los resultados de la cuarta query

3. En el root del proyecto, crear una carpeta bin.
    - Al ejecutar el comando de setup, en esta carpeta se va a generar un script de bash (*maniac.bash*) que permite probar la tolerancia a fallos del sistema.
    - Este script se encarga de, cada 10 segundos, eliminar la mitad del total de contenedores del sistema
    
### Ejecutar el sistema

Para ejecutar el sistema es necesario correr los siguientes comandos:

1. ```bash
        make build-image
    ```
    - Este comando construye las imagenes de docker.
2. ```bash
        Q1=N Q2=M Q3=P Q4=Q HB=X make setup 
    ```
    - Este comando construye el archivo de docker compose. Se puede ejecutar sin necesidad de setear las variables. Pero al setear las variables se puede cambiar la cantidad de workers para cada una de las queries.
        - Q1 $\longrightarrow$ Setea la cantidad de workers para la Query por cantidad de escalas que tendra el sistema
        - Q2 $\longrightarrow$ Setea la cantidad de workers por distancia total recorrida que tendra el sistema 
        - Q3 $\longrightarrow$ Setea la cantidad de workers por tiempo total que tendra el sistema
        - Q4 $\longrightarrow$ Setea la cantidad de workers de filtro por promedio que tendra el sistema
        - HB $\longrightarrow$ Setea la cantidad de workers que ejecutan el control y el re-instanciamiento de contenedores caidos que tendra el sistema
    - Se debe asegurar que no existe un documento *docker-compose-dev.yaml* para que este comando pueda ejecutarse correctamente.
3. ```bash
        make docker-compose-up
    ```
    - Este comando crea los contenedores y levanta el sistema con la configuracion previamente establecida.
4. ```bash
        make run-client
    ```
    - Este comando ejecuta el cliente