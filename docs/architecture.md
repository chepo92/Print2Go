┌──────────────────────────┐
│     UI / Controlador     │
│ (botón stop, pause, etc) │
└────────────┬─────────────┘
             │
             ▼
┌──────────────────────────┐
│ SerialManager            │
│                          │
│  ┌──────────┐            │
│  │ priorityQ│◄─ STOP     │  
│  └──────────┘            │
│        │                 │
│        ▼                 │
│     sendQ (normal)       │
│                          │
└────────────┬─────────────┘
             │
             ▼
┌──────────────────────────┐
│ writeLoop                │
│ - prioridad primero      │ 
│ - ACK correcto           │
└────────────┬─────────────┘
             │
             ▼
┌──────────────────────────┐
│ Firmware impresora       │
│ (Marlin)                 │
└──────────────────────────┘



┌──────────────────────────┐
│     UI / Controlador     │
│ (botón stop, pause, etc) │
└────────────┬─────────────┘
             │
             ▼
┌──────────────────────────┐
│        WebAPI            │
│ (webapi package)         │
│                          │
│ - Endpoints REST / AJAX  │
│ - Manejo de uploads      │
│ - Status JSON para UI    │
│ - Llama a JobManager     │
└─────────────┬────────────┘
              │
              │ llama
              ▼
┌──────────────────────────┐
│      JobManager          │
│ (job package)            │
│                          │
│ - Controla Jobs de print │
│ - Mantiene TaskStatus    │
│ - Subscribe/Notify UI    │
│ - Maneja cancel/shutdown │
│ - Llama a SerialManager  │
└─────────────┬────────────┘
              │
              │ llama / envía gcode
              ▼
┌──────────────────────────┐
│    SerialManager         │
│ (serial/serialmgr)       │
│                          │
│ - Control de puerto serie│
│ - Envía comandos GCode   │
│ - Recibe feedback        │
│ - Reenvío de errores     │
└─────────────┬────────────┘
              │
              │ usa
              ▼
┌──────────────────────────┐
│      Printer HW          │
│ (impresora 3D real)      │
│                          │
│ - Ejecuta GCode          │
│ - Devuelve ACK / errores │
└──────────────────────────┘



