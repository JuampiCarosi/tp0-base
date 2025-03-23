# TP0: Docker + Comunicaciones + Concurrencia

## Parte 1: Introducción a Docker

### Ejercicio N°1:

Se creó el archivo `generar-compose.sh` que genera el archivo `docker-compose.yml` en root del proyecto con el nombre pasado como primer argumento y la cantidad de clientes pasada como segundo argumento. El script ejecuta un script en go ubicado en la carpeta `scripts` que genera el archivo `docker-compose.yml`, el script de bash le pasa los argumentos al de go y este ultimo es el que se encarga de generar el archivo `docker-compose.yml`.

### Ejercicio N°2:

Se utilizó un bind mount para montar el archivo de configuración del cliente y el servidor en el container. En ambos casos se utilizó el flag `--exclude` de la beta `# syntax=docker/dockerfile:1.7-labs` para que no se copiaran los archivos de configuración a la imagen.

### Ejercicio N°3:

Para el `validar-echo-server.sh` se utilizó el comando `docker run` sobre una imagen de alpine que tiene netcat instalado. Se corrió el comando `echo "test" | nc server 12345` para verificar que el servidor esté funcionando respondiendo 'test'.

### Ejercicio N°4:

Se modifico el servidor para que al recibir la signal SIGTERM, cierre la conexion actual y el socket. Ademas de eso setea el atributo `running` en False para que cuando el accept se desbloquee (ya que se cerro el socket) se pueda salir del while y terminar el programa.

El cliente para que termine de forma _graceful_ crea una go routine que escucha la signal SIGTERM, al recibirla el cliente cierra el loop y deja de enviar mensajes, y si habia una conexion con el servidor la cierra.

## Parte 2: Repaso de Comunicaciones

Para esta seccion se reescribio el codigo del servidor en go como desafio personal para practicar el lenguaje y poder aprovechar sus estructuras de concurrencia.

### Ejercicio N°5:

#### Protocolo de comunicación:

Se define un protocolo de comunicación para el envío y la recepción de los paquetes, el mismo es implementado en el archivo `shared/communication.go`. Para los mensajes se creo una interfaz `Message` que define como se serializa y deserializa el mensaje.

El protocolo consiste en un header fijo de 8 bytes que los primeros 4 bytes indican el tipo de mensaje y los ultimos 4 bytes indican el largo del payload. El resto del mensaje es el payload.

Para realizar la tarea de recepcion de mensaje se creo una funcion `MessageFromSocket` que recibe un socket y devuelve un mensaje con el tipo, el largo del payload y el payload. Por lo que la funcion `Deserialize` de la interfaz `Message` simplemente recibe el payload y lo deserializa segun el tipo de mensaje.

De esta forma como se serializa y deserializa cada mensaje en especifico depende de la implementacion de la interfaz `Message` para cada tipo de mensaje.

#### Serialización para BetMessage:

La serialización para el mensaje de apuesta se encarga de serializar los datos de la apuesta en un string separado por `;`, con el formato `agencia;nombre;apellido;dni;nacimiento;numero`

#### Serialización para BetResponse:

La serialización para el mensaje de respuesta se encarga de serializar el booleano en un string "SUCCESS" o "ERROR" .

### Ejercicio N°6:

Para este ejercicio se creo un mensaje de tipo `BatchBetMessage` que contiene un pseudo csv de apuestas, siguiendo el mismo formato que el del ejercicio 5, solo que cada apuesta esta separada por un salto de linea de la siguiente.

Cada Batch de apuestas se va cargando en memoria a medida que se necesita enviar un batch al servidor, para esto se utiliza un reader de csv que se va recorriendo en el metodo `LoadAgencyBatch` de la estructura `Client`.

Para el valor por defecto de la cantidad de apuestas por batch se asumio que cada record del csv contiene como maximo 70 caracteres (ya que es una cantidad razonable teniendo en cuenta el largo del nombre, puesto a que ninguno de los ejemplos supera esta cifra). De esta manera para cumplir con los 8Kb por defecto se setea el valor de `batch.maxAmount` en 105.

### Ejercicio N°7:

Para este ejercicio se creo un map en el servidor que guarda las agencias que han enviado todos sus datos. Cuando el servidor recibe el mensaje de que todas las agencias han enviado sus datos, se procede procesar los ganadores del sorteo. Una vez que se tienen todos los ganadores recien ahi los clientes pueden consultar los ganadores del sorteo. Si se realiza una consulta antes de que el servidor tenga todos los ganadores, el servidor responde con un mensaje que no estan los ganadores todavia.

Los clientes tienen 10 intentos para consultar los ganadores, esperando 100ms entre cada intento.

## Parte 3: Repaso de Concurrencia

En este ejercicio es importante considerar los mecanismos de sincronización a utilizar para el correcto funcionamiento de la persistencia.

### Ejercicio N°8:

Para implementar la concurrencia en primer lugar se convirtio la llamada a `handleClientConnection` en una go routine para que se ejecute en paralelo con el accept del socket. Para esto se tuvieron que hacer modificaciones en el struct server para poder guardar multiples conexiones y poder cerrarlas todas en el metodo `Shutdown`, todo de forma thread safe.

Las principales modificaciones fueron:

- Transformar el campo de conexion a un map de conexiones y agregar un mutex para poder acceder y modificar el map de forma thread safe.
- Transformar el metodo `Shutdown` para que cierre todas las conexiones del map.
- Transformar el campo de receivedAgencies a un channel, donde va a haber una go routine esperando a que todas las agencias notifiquen que han enviado sus datos, y una vez que lo hace escribe en el `Server.winners` los ganadores de cada agencia.
- Crear el campo `Server.betsMutex` para poder manejar la lectura y escritura de los datos de las apuestas ya que estas operaciones no son thread safe. Por mas que `LoadBets` no deberia llamarse a la vez que `SaveBets` u otro `LoadBets`, se opto por igualmente hacerlo thread safe pensando en que en un futuro podrian empezar a recibirse mas apuestas mientras se realiza un sorteo.

## Condiciones de Entrega

Se espera que los alumnos realicen un _fork_ del presente repositorio para el desarrollo de los ejercicios y que aprovechen el esqueleto provisto tanto (o tan poco) como consideren necesario.

Cada ejercicio deberá resolverse en una rama independiente con nombres siguiendo el formato `ej${Nro de ejercicio}`. Se permite agregar commits en cualquier órden, así como crear una rama a partir de otra, pero al momento de la entrega deberán existir 8 ramas llamadas: ej1, ej2, ..., ej7, ej8.
(hint: verificar listado de ramas y últimos commits con `git ls-remote`)

Se espera que se redacte una sección del README en donde se indique cómo ejecutar cada ejercicio y se detallen los aspectos más importantes de la solución provista, como ser el protocolo de comunicación implementado (Parte 2) y los mecanismos de sincronización utilizados (Parte 3).

Se proveen [pruebas automáticas](https://github.com/7574-sistemas-distribuidos/tp0-tests) de caja negra. Se exige que la resolución de los ejercicios pase tales pruebas, o en su defecto que las discrepancias sean justificadas y discutidas con los docentes antes del día de la entrega. El incumplimiento de las pruebas es condición de desaprobación, pero su cumplimiento no es suficiente para la aprobación. Respetar las entradas de log planteadas en los ejercicios, pues son las que se chequean en cada uno de los tests.

La corrección personal tendrá en cuenta la calidad del código entregado y casos de error posibles, se manifiesten o no durante la ejecución del trabajo práctico. Se pide a los alumnos leer atentamente y **tener en cuenta** los criterios de corrección informados [en el campus](https://campusgrado.fi.uba.ar/mod/page/view.php?id=73393).
