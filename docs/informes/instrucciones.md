## Instrucciones de uso

### Configuracion del cliente

Para configurar el cliente es necesario seguir los siguientes pasos:

1. En la carpeta `clients/`, agregar un directorio con la siguiente estructura:
   los archivos `test.csv` y `airports-codepublic.csv` en el subdirectorio
   data. Estos archivos contienen la informacion de los vuelos y las
   coordenadas de los aeropuertos respectivamente. El archivo `config.yaml` en
   el sub-directorio `config`.
   Ejemplo:

   ```sh
   $ tree clients/

   clients/
   └── c1
       ├── config
       │   └── config.yaml
       └── data
           ├── airports-codepublic.csv
           └── test.csv

   4 directories, 3 files
   ```

2. En la carpeta `results` en el sub-directorio de `clients/` escogido se
   guardan los resultados de las 4 consultas luego de una corrida del cliente.
    - first.csv contiene los resultados de la primer consulta.
    - second.csv contiene los resultados de la segunda consulta.
    - third.csv contiene los resultados de la tercera consulta.
    - fourth.csv contiene los resultados de la cuarta consulta.

3. Al ejecutar el comando de setup, en la carpeta `bin/` se va a generar un
   script de bash (*maniac.bash*) que permite probar la tolerancia a fallos del
   sistema.
    - Este script se encarga de, cada 10 segundos, eliminar la mitad del total
      de contenedores del sistema (o la cantidad configurada).

### Ejecutar el sistema

Para ejecutar el sistema es necesario correr los siguientes comandos:

1. ```bash
    make build-image
    ```
    - Este comando construye las imagenes de docker.

2. ```bash
    Q1=N Q2=M Q3=P Q4=Q HB=X make setup
    ```
    - Este comando construye el archivo de docker compose. Se puede ejecutar
      sin necesidad de setear las variables. Pero al setear las variables se
      puede cambiar la cantidad de workers para cada una de las queries.
        + Q1 $\longrightarrow$ Setea la cantidad de workers para la Query por
          cantidad de escalas que tendra el sistema
        + Q2 $\longrightarrow$ Setea la cantidad de workers por distancia total
          recorrida que tendra el sistema 
        + Q3 $\longrightarrow$ Setea la cantidad de workers por duración que
          tendra el sistema
        + Q4 $\longrightarrow$ Setea la cantidad de workers de filtro por
          promedio que tendra el sistema
        + HB $\longrightarrow$ Setea la cantidad de workers que ejecutan el
          control y el re-instanciamiento de contenedores caidos que tendra el
          sistema

3. ```bash
    make docker-compose-up
    ```
    - Este comando crea los contenedores y levanta el sistema con la
      configuracion previamente establecida.

4. ```bash
    make run-client
    ```
    - Este comando ejecuta el cliente, si se escogio un nombre distinto de
      `c1/` (como se ve en el ejemplo) para el directorio del cliente, se debe
      indicar setenado la variable de entorno \$CLIENT_ID=<directorio>.

5. En caso de querer probar la tolerancia a fallos del sistema, se debe
   ejecutar el siguiente comando con la terminal sobre el root del proyecto:

```bash
bin/maniac.bash
```
