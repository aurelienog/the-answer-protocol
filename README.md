## Behavioral design Patterns (GoF)

### **Command** — ejecutar acciones (main design pattern)

**Encapsula cada comando del protocolo en un handler independiente.**

Problema que resuelve: el servidor debe interpretar y ejecutar muchos comandos diferentes: MOVE, CHAT, TAKE, ATTACK, QUEST, etc.
En lugar de escribir un gran bloque switch o una cadena de if/else con toda la lógica mezclada, cada comando se representa mediante un objeto o estructura que encapsula su ejecución.

Por ejemplo:
- MoveCommand
- TakeCommand
- AttackCommand
- ChatCommand
- QuestCommand

Todos siguen una interfaz común, como execute(context).

### **Observer** — notificar acontecimientos (complement design pattern)

**Permite que los componentes interesados reciban notificaciones cuando sucede algo en el mundo.**
Se utiliza cuando el servidor debe informar de cambios sin esperar a que el cliente los solicite.

Problema que resuelve: cuando un jugador se mueve, recoge un objeto o entra en combate, otros jugadores deben recibir eventos sin tener que solicitar continuamente el estado del mundo.
El mundo o los componentes que gestionan el estado publican eventos, y un sistema de notificaciones los distribuye a los clientes afectados.

Por ejemplo:
- PlayerEnteredRoom
- PlayerLeftRoom
- ChatMessageReceived
- ItemTaken
- CombatResult

Esto resulta especialmente útil para mantener sincronizados los clientes CLI y GUI.


### example
```
Imaginad que Alice ejecuta MOVE north.
1. Cliente web
Envía MOVE north por TCP

2. Command Dispatcher
Selecciona el handler MOVE

3. World / Movement Service
Mueve a Alice y actualiza el estado

4. Event Bus (Observer)
Publica PlayerLeftRoom y PlayerEnteredRoom

Habitación anterior
Notifica que Alice se ha ido


Habitación nueva
Notifica que Alice ha llegado
```

Los eventos se traducen al formato definido por vuestro RFC y se envían a los sockets TCP de los jugadores afectados. El cliente web recibe esos mensajes y actualiza la interfaz.

### State — estados de combate y jugador (Opcional)

Problema que resuelve: el comportamiento de un jugador cambia según su situación: vivo, en combate, derrotado o reapareciendo.
En lugar de dispersar comprobaciones como if (hp == 0) por todo el código, puedes centralizar las transiciones de estado.
Es particularmente útil si añadís acciones como DEFEND o FLEE, aunque no es necesario crear una jerarquía de clases compleja para un sistema de combate sencillo.


### ¿Por qué es especialmente adecuado para TAP?

| Requisito del proyecto	| Patrón o componente |	Ventaja | 
| --- | --- | --- | 
| Implementar todos los comandos RFC	| Command + Dispatcher	| Cada operación se puede probar de forma independiente| 
| Actualizar a los jugadores en tiempo real	| Observer / Event Bus	| Las notificaciones no quedan mezcladas con la lógica del juego| 
| Gestionar combate y reaparición	| State, si la complejidad lo justifica	| Centraliza transiciones y restricciones | 
| Cargar el mundo desde YAML/JSON	| Repository + validación	| Separa los datos estáticos del estado dinámico| 
| Registrar comandos, errores y cambios	| Logger estructurado	| Facilita depuración y auditoría| 
| Mantener intercambiables CLI y GUI	| Arquitectura orientada al protocolo	| Ambos clientes utilizan el mismo contrato de comunicación | 

Hay un detalle importante: el patrón Observer no debe confundirse con el envío directo de mensajes TCP. El Observer organiza la generación y distribución lógica de eventos; otro componente se encarga de serializarlos en el formato RFC y enviarlos a los sockets correspondientes.

Asimismo, el servidor debe controlar las modificaciones del estado compartido para evitar duplicaciones de objetos, por ejemplo, cuando dos jugadores intentan ejecutar TAKE sobre el mismo objeto. Command no resuelve automáticamente la concurrencia: el servicio de inventario y el modelo del mundo deben garantizar esa consistencia.

### ¿Qué implementaría realmente?

Mi recomendación para un proyecto académico de este tamaño sería priorizar solo estas piezas:
- Command + Dispatcher: para todos los comandos del protocolo.
- Observer o Event Bus sencillo: para los eventos de presencia, chat y combate.
- World Repository: para cargar y validar el mundo estático.
- Game Services: para mantener separadas las reglas de movimiento, inventario, combate y misiones.
Dejaría el patrón State como opcional y no introduciría patrones más complejos, como Abstract Factory o una jerarquía extensa de estrategias, salvo que el diseño real lo necesite.

## Architecture

1. tree

```
the-answer-protocol/
│
├── cmd/                          # Puntos de entrada ejecutables
│   ├── server/
│   │   └── main.go               # Arranca el servidor del juego
│   ├── cli/
│   │   └── main.go               # Arranca el cliente CLI
│   └── webgateway/
│       └── main.go               # Puente HTTP/WebSocket para el navegador
│
├── internal/                     # Código privado de la aplicación
│   │
│   ├── server/
│   │   ├── server.go             # Ciclo de vida del servidor TCP
│   │   ├── connection.go         # Gestión de cada conexión
│   │   └── session.go            # Sesión y jugador conectado
│   │
│   ├── protocol/                 # RFC 42TAP
│   │   ├── parser.go             # Analiza mensajes recibidos
│   │   ├── encoder.go            # Serializa respuestas y eventos
│   │   ├── messages.go           # Tipos de mensajes
│   │   └── errors.go             # Errores del protocolo
│   │
│   ├── commands/                 # Command Pattern
│   │   ├── dispatcher.go         # Registro y distribución
│   │   ├── connect.go
│   │   ├── look.go
│   │   ├── move.go
│   │   ├── chat.go
│   │   ├── take.go
│   │   ├── drop.go
│   │   ├── inventory.go
│   │   ├── talk.go
│   │   ├── attack.go
│   │   ├── status.go
│   │   ├── quest.go
│   │   ├── quests.go
│   │   ├── who.go
│   │   ├── group.go
│   │   └── quit.go
│   │
│   ├── game/                     # Dominio y reglas del mundo
│   │   ├── world.go              # Estado global del mundo
│   │   ├── room.go               # Habitaciones y salidas
│   │   ├── player.go             # Jugadores y HP
│   │   ├── item.go               # Instancias de objetos
│   │   ├── npc.go                # NPCs y estadísticas
│   │   ├── inventory.go          # Gestión del inventario
│   │   ├── movement.go           # Reglas de movimiento
│   │   ├── combat.go             # Sistema de combate
│   │   └── quest.go              # Misiones y recompensas
│   │
│   ├── events/                   # Observer / Event Bus
│   │   ├── event.go              # Definición de eventos
│   │   ├── bus.go                # Publicación y suscripción
│   │   └── router.go             # A qué jugadores enviar eventos
│   │
│   ├── worlddata/                # Carga y validación del mundo
│   │   ├── loader.go             # Lee YAML/JSON
│   │   └── validate.go           # Valida IDs, salidas y referencias
│   │
│   ├── gateway/                  # Integración con navegador
│   │   ├── http.go
│   │   └── websocket.go
│   │
│   └── logging/
│       └── logger.go             # Logs estructurados JSON
│
├── clients/
│   ├── cli/
│   │   └── README.md             # Diseño del cliente CLI
│   └── web/
│       ├── index.html
│       ├── src/
│       │   ├── app.js
│       │   ├── network.js        # Comunicación WebSocket
│       │   ├── state.js          # Estado de la interfaz
│       │   └── ui.js             # Renderizado de la GUI
│       └── styles/
│           └── main.css
│
├── data/
│   ├── world.yaml                # Habitaciones, objetos y NPCs
│   └── quests.yaml               # Misiones, si se separan del mundo
│
├── tests/
│   ├── integration/
│   └── protocol/
│
├── docs/
│   ├── architecture.md
│   └── protocol-decisions.md
│
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── .gitignore
```

2. Qué responsabilidad tiene cada carpeta

| Carpeta	| Responsabilidad	| ¿Conoce el RFC? | 
| --- | --- | --- |
| server/	| Aceptar conexiones y gestionar sesiones	| Sí, a través de protocol/ | 
| protocol/	| Interpretar y generar mensajes válidos	| Sí, directamente | 
| commands/	| Ejecutar las órdenes recibidas	| Utiliza el protocolo, pero delega las reglas | 
| game/	| Aplicar las reglas del mundo | Idealmente, no |
| events/	| Distribuir acontecimientos del juego	| Conoce los eventos internos | 
| worlddata/	| Cargar y validar los datos estáticos	| No necesita conocer los clientes | 
| gateway/	| Traducir entre WebSocket y el protocolo del juego	| Sí, como adaptador | 
| logging/	| Registrar operaciones y errores	| Puede recibir contexto de todas las capas | 

## Start

1. 

Cómo preparar el repositorio por fases
No empezaría implementando los 16 comandos a la vez. Crearía primero la estructura y haría que el recorrido básico funcionase de extremo a extremo.

Fase 1 — Esqueleto funcional
Crear go.mod, Makefile, cmd/server, internal/server e internal/protocol. Conseguir que el servidor acepte conexiones TCP y lea mensajes terminados en \n.




Fase 2 — Command y mundo
Implementar CONNECT, LOOK y MOVE; añadir el dispatcher y cargar un mapa desde YAML. Después incorporar inventario, chat, NPCs, combate y misiones.




Fase 3 — Observer y concurrencia
Añadir eventos de presencia y chat, garantizar que una conexión lenta no bloquee las demás y gestionar las desconexiones limpiamente.




Fase 4 — CLI y navegador
Conectar el CLI al TCP y la GUI mediante el gateway WebSocket. Verificar que ambos clientes pueden jugar simultáneamente y reciben los mismos eventos del mundo.



2. 

Dos decisiones que os ahorrarán problemas
- Un único estado autoritativo del mundo: los objetos, las posiciones de los jugadores y los HP se gestionan en el servidor. Usad un mecanismo claro de sincronización en Go para evitar carreras de datos, por ejemplo, un actor que serialice las operaciones del mundo o un sync.Mutex bien delimitado.
- El protocolo es un contrato compartido: CLI, gateway y servidor deben respetar las mismas reglas RFC. No hagáis que el cliente web invente formatos alternativos para comandos del juego.
Y una recomendación adicional: en Go, evitad crear demasiados paquetes pequeños que dependan circularmente unos de otros. La estructura anterior es un mapa conceptual; si dos archivos forman una misma responsabilidad y no necesitan reutilización independiente, pueden estar en el mismo paquete.

3.

Mi propuesta para empezar hoy: cread cmd/server, cmd/cli, internal/protocol, internal/server, internal/commands, internal/game, internal/events, data/ y clients/web/. El resto puede añadirse cuando exista una necesidad real.

## Web client

1. Svelte + Tailwind CSS
Mi favorita para vuestro proyecto
Interfaz ligera y reactiva, ideal para que las habitaciones, los contadores de jugadores, el inventario y el chat se actualicen al recibir eventos.
- Diseño: muy flexible y fácil de personalizar.
- Animaciones: transiciones de paneles, notificaciones y cambios de estado.
- Complejidad: moderada-baja.
- Integración: WebSocket con el gateway Go.
Elegiría Svelte si queréis una GUI atractiva sin invertir demasiado tiempo en infraestructura frontend.

2. React + TypeScript + Tailwind CSS
Una opción excelente si queréis muchos componentes reutilizables y una estructura frontend conocida.
- Diseño: gran flexibilidad y ecosistema amplio.
- Animaciones: podéis añadir Motion para transiciones y microinteracciones.
- Complejidad: moderada.
- Integración: muy adecuada para gestionar mensajes WebSocket y componentes independientes.
La elegiría si alguno del grupo ya conoce React o queréis una interfaz

### Distribution:

La distribuiría en cinco zonas:
- Zona central: nombre, descripción e ilustración de la habitación actual, con botones para las salidas disponibles.
- Panel izquierdo: mapa de habitaciones exploradas y jugadores presentes.
- Panel derecho: personaje, HP, inventario, objetos disponibles y misiones.
- Parte inferior: chat global, chat de habitación y chat de grupo, separados de los logs del sistema.
- Notificaciones flotantes: objetos recogidos, daño recibido, jugadores que entran y misiones completadas.

Para las animaciones, usaría transiciones cortas y discretas, indicadores de vida animados, notificaciones temporales y efectos visuales al recoger objetos. No intentaría convertir cada acción en una animación compleja: la información debe seguir siendo legible mientras llegan eventos en tiempo real.

### Stack recomendado

- Backend
Go · TCP · RFC 42TAP · Command · Event Bus

- Comunicación web
WebSocket · gateway Go

- Frontend
Svelte + TypeScript + Tailwind CSS

- Estética
RPG retro, paneles oscuros, tipografía legible y animaciones sutiles

## UX

### 1. Pilares de UX (Experiencia de Usuario)

- Feedback Inmediato (Game Feel): Cada toque (tap) en la pantalla debe generar una respuesta visual o háptica instantánea (vibración sutil, destellos de partículas, o textos flotantes) para que el jugador sienta el peso de sus decisiones o acciones rítmicas.
- Carga Cognitiva Reducida: Como la narrativa exige leer o escuchar diálogos, los botones de acción e interfaces cooperativas/competitivas no deben obstruir el área de texto principal.
- Sincronización Multijugador Visible: Al interactuar simultáneamente con otros jugadores, es vital ver en tiempo real qué están eligiendo o cuántos clics están aportando a un evento cooperativo rápido (Quick Time Event).

| Elemento de UI	| Tipo de Interfaz	| Función en un juego narrativo de "TAP"| 
| --- | --- | --- |
| Indicadores Co-op	| Espacial / No diegética	| Pequeños avatares o barras de progreso que muestran los taps por segundo (TPS) acumulados por el grupo para superar un obstáculo en la historia.| 
| Botones de Decisiones| 	No diegética	| Aparecen en la mitad inferior de la pantalla (zona óptima para el alcance del pulgar). Al mantener presionado o hacer tap, el jugador vota o ejecuta una ruta narrativa.| 


Dinámicas Multijugador aplicadas al "TAP"

• Taps de Votación Democrática: Cuando la historia se bifurca, los jugadores hacen tap repetidamente sobre su opción favorita. El botón se va llenando de color según el volumen de clics grupales, decidiendo la ruta de forma dinámica en pocos segundos.
• QTEs Cooperativos (Quick Time Events): Para momentos de tensión narrativa (por ejemplo, empujar una puerta juntos para escapar de un peligro), aparece un gran botón central en el que todo el lobby debe sumar una cantidad meta de taps antes de que expire el temporizador.
• El "Tap" Silencioso (Interrupción): Una mecánica en la que un jugador puede presionar un botón de "Interrumpir" en cualquier momento de un diálogo para cambiar drásticamente el rumbo de la conversación, generando un giro inesperado para el resto de los participantes.


### 2. Flujos de UX Críticos para Multijugador

| Situación de Juego | Solución de UX Recomendada | 
| --- | --- |
| Jugadores leyendo a ritmos diferentes	| - Implementar un botón de "Listo" individual. La historia no avanza hasta que la mayoría o el 100% hayan pulsado el botón. - Implementar temporizadores dinámicos (timers) visibles pero no intrusivos que automaticen o fuercen una decisión si alguien se desconecta o tarda demasiado. - La UX debe evitar frustracion implementando temporizadores visibles pero organicos para las votaciones o elecciones de dialogo. - Mostrar de forma sutil qué opción está eligiendo cada jugador (usando avatares pequeños o bordes de color asignados a cada usuario en tiempo real).| 
| Desconexión de un jugador	| La UI debe notificar al grupo y activar un modo de "decisión por IA" o delegar el voto para no congelar la partida. - | Jugadores ausentes o lentos	| Implementar temporizadores dinámicos (timers) visibles pero no intrusivos que automaticen o fuercen una decisión si alguien se desconecta o tarda demasiado. | 
| Conflictos de opinión (Empates)	| Diseñar mecánicas transparentes de desempate: dados virtuales visibles por todos, un sistema de "puntos de convicción" acumulables para forzar decisiones, o aleatoriedad dramática.| 
| Comunicación integrada	| Chat de texto flotante, rueda de emojis rápidos (para expresar reacciones a la trama) o canales de voz integrados con indicadores visuales de quién habla.| 
| Historial de la trama	| Un diario o "Log" accesible en un clic para revisar decisiones pasadas y las consecuencias que el grupo eligió.| 

### 3. Sistemas de Comunicación e Información Asíncrona

Los juegos narrativos dependen de la atmósfera y el ritmo. La UI de comunicación no debe saturar la pantalla.
• Sistema de "Ping" Contextual: Al estilo de juegos cooperativos modernos, permitir que un jugador señale un objeto del entorno para abrir una línea de diálogo o debate narrativo sin necesidad de escribir en un chat de texto.
• Historial de Decisiones (Log): Un menú accesible que guarde la línea de tiempo de lo que se ha decidido y qué jugador influyó en cada giro argumental. Esto ayuda a reengancharse si alguien se distrae.
• Estados de Ánimo y Relaciones: Indicadores visuales abstractos (como cambios rápidos de color o micro-animaciones en los retratos de los personajes) para alertar si la decisión de un compañero afectó la confianza de un NPC.

### 4. Accesibilidad 

- Configuración de texto robusta: Subtítulos altamente personalizables (tamaño, color de fondo, indicadores de quién está hablando) indispensables para el juego remoto y sin audio.
- font sans serif, alto contraste


