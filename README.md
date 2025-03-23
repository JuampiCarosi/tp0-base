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

Modificar los clientes para que envíen varias apuestas a la vez (modalidad conocida como procesamiento por _chunks_ o _batchs_).
Los _batchs_ permiten que el cliente registre varias apuestas en una misma consulta, acortando tiempos de transmisión y procesamiento.

La información de cada agencia será simulada por la ingesta de su archivo numerado correspondiente, provisto por la cátedra dentro de `.data/datasets.zip`.
Los archivos deberán ser inyectados en los containers correspondientes y persistido por fuera de la imagen (hint: `docker volumes`), manteniendo la convencion de que el cliente N utilizara el archivo de apuestas `.data/agency-{N}.csv` .

En el servidor, si todas las apuestas del _batch_ fueron procesadas correctamente, imprimir por log: `action: apuesta_recibida | result: success | cantidad: ${CANTIDAD_DE_APUESTAS}`. En caso de detectar un error con alguna de las apuestas, debe responder con un código de error a elección e imprimir: `action: apuesta_recibida | result: fail | cantidad: ${CANTIDAD_DE_APUESTAS}`.

La cantidad máxima de apuestas dentro de cada _batch_ debe ser configurable desde config.yaml. Respetar la clave `batch: maxAmount`, pero modificar el valor por defecto de modo tal que los paquetes no excedan los 8kB.

Por su parte, el servidor deberá responder con éxito solamente si todas las apuestas del _batch_ fueron procesadas correctamente.

### Ejercicio N°7:

Modificar los clientes para que notifiquen al servidor al finalizar con el envío de todas las apuestas y así proceder con el sorteo.
Inmediatamente después de la notificacion, los clientes consultarán la lista de ganadores del sorteo correspondientes a su agencia.
Una vez el cliente obtenga los resultados, deberá imprimir por log: `action: consulta_ganadores | result: success | cant_ganadores: ${CANT}`.

El servidor deberá esperar la notificación de las 5 agencias para considerar que se realizó el sorteo e imprimir por log: `action: sorteo | result: success`.
Luego de este evento, podrá verificar cada apuesta con las funciones `load_bets(...)` y `has_won(...)` y retornar los DNI de los ganadores de la agencia en cuestión. Antes del sorteo no se podrán responder consultas por la lista de ganadores con información parcial.

Las funciones `load_bets(...)` y `has_won(...)` son provistas por la cátedra y no podrán ser modificadas por el alumno.

No es correcto realizar un broadcast de todos los ganadores hacia todas las agencias, se espera que se informen los DNIs ganadores que correspondan a cada una de ellas.

## Parte 3: Repaso de Concurrencia

En este ejercicio es importante considerar los mecanismos de sincronización a utilizar para el correcto funcionamiento de la persistencia.

### Ejercicio N°8:

Modificar el servidor para que permita aceptar conexiones y procesar mensajes en paralelo. En caso de que el alumno implemente el servidor en Python utilizando _multithreading_, deberán tenerse en cuenta las [limitaciones propias del lenguaje](https://wiki.python.org/moin/GlobalInterpreterLock).

## Condiciones de Entrega

Se espera que los alumnos realicen un _fork_ del presente repositorio para el desarrollo de los ejercicios y que aprovechen el esqueleto provisto tanto (o tan poco) como consideren necesario.

Cada ejercicio deberá resolverse en una rama independiente con nombres siguiendo el formato `ej${Nro de ejercicio}`. Se permite agregar commits en cualquier órden, así como crear una rama a partir de otra, pero al momento de la entrega deberán existir 8 ramas llamadas: ej1, ej2, ..., ej7, ej8.
(hint: verificar listado de ramas y últimos commits con `git ls-remote`)

Se espera que se redacte una sección del README en donde se indique cómo ejecutar cada ejercicio y se detallen los aspectos más importantes de la solución provista, como ser el protocolo de comunicación implementado (Parte 2) y los mecanismos de sincronización utilizados (Parte 3).

Se proveen [pruebas automáticas](https://github.com/7574-sistemas-distribuidos/tp0-tests) de caja negra. Se exige que la resolución de los ejercicios pase tales pruebas, o en su defecto que las discrepancias sean justificadas y discutidas con los docentes antes del día de la entrega. El incumplimiento de las pruebas es condición de desaprobación, pero su cumplimiento no es suficiente para la aprobación. Respetar las entradas de log planteadas en los ejercicios, pues son las que se chequean en cada uno de los tests.

La corrección personal tendrá en cuenta la calidad del código entregado y casos de error posibles, se manifiesten o no durante la ejecución del trabajo práctico. Se pide a los alumnos leer atentamente y **tener en cuenta** los criterios de corrección informados [en el campus](https://campusgrado.fi.uba.ar/mod/page/view.php?id=73393).
