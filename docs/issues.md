Creo que no está esperando en ciertos casos, por ejemplo el firmware stock de ender, al estar moviendo por ejemplo G28 enviara
echo:busy: processing
mientras este procesando, tambien enviara:
X:0.00 Y:0.00 Z:0.00 E:0.00 Count X:0 Y:0 Z:0
y finalmente enviara
ok


Si se le envia M105, respondera:
ok T:16.25 /0.00 B:16.09 /0.00 @:0 B@:0

M104 S200, responde con
ok

Algunos M109 S100, responde con 
 T:99.94 /100.00 B:16.21 /0.00 @:17 B@:0 W:4
 T:99.67 /100.00 B:16.13 /0.00 @:25 B@:0 W:3
echo:busy: processing
 T:99.33 /100.00 B:16.17 /0.00 @:34 B@:0 W:2
 T:99.04 /100.00 B:16.13 /0.00 @:41 B@:0 W:1
echo:busy: processing
 T:98.80 /100.00 B:16.21 /0.00 @:47 B@:0 W:0
ok

To Do

Backend
- [ ] Save and persist config
- [ ] Agregar timeout para cuando no se realiza la conexion inicial y se queda pegada
- [ ] Detect stale print
- [x] Read and report temps
- [ ] Save file
- [ ] Timelapse 

Frontend
- [ ] Mordernize Js
- [ ] Modularize functions
- [ ] Migrate to Vue 3
- [ ] Add controls
- [ ] File search 
- [ ] Control for individual and custom setpoint of temps
- [ ] Gcode Viewer




