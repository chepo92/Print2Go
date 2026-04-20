- Install openwrt

- Configure network, decide if the final setup will use ethernet or wifi: 
LAN: 
    - Conect LAN cable to a pc
    - Configure LAN ip to something available in your main network (or set to dynamic and set a fixed ip for the box on your main router)
Wifi
    - Connect via LAN to configure
    - Change box LAN IP to be a different network from your main routers (eg. main is 192.168.1.1, config box LAN to 192.168.8.1)
    - Connect as client, Scan for wifi and set password, add to "wwan" network. The scan may temporally (1 min) disconnect you from the box. 
    - In firewall, add and configure "wwan" to "lan" and accept in and out.
    - Save and apply
    - Configure WLAN ip to something available in your main network (or set to dynamic and set a fixed ip for the box on your main router)
    
- Apply extroot :

(thanks octoWRT/klipperWRT)

- Copy/send Print2Go and Print2Go-srv: 

Send to device, put the Print2Go binary in usr/bin, 
scp -O Print2Go root@192.168.8.1:/usr/bin

Copy the Print2Go-srv /etc/init.d  autostart script in etc/init.d folder
scp -O Print2Go-srv root@192.168.8.1:/etc/init.d

(Optional copy back)
scp -O root@192.168.8.1:/etc/init.d/Print2Go-srv Print2Go-srv-linux

Enter ssh as root
ssh root@192.168.8.1

Make the autostart and the bin executable

chmod +x /etc/init.d/Print2Go-srv
chmod +x /usr/bin/Print2Go

Test it 
/usr/bin/Print2Go -tty /dev/ttyUSB0 -listen 192.168.8.1:5001

Run full headless:
nohup /usr/bin/Print2Go -tty /dev/ttyUSB0 -listen 192.168.8.1:5001 >/dev/null 2>&1 &

Run as daemon (stops if terminal is closed):
/usr/bin/Print2Go -tty /dev/ttyUSB0 -listen 192.168.8.1:5001 >/dev/null 2>&1 &

New (no need usb port)
nohup ./Print2Go -listen 192.168.8.1:5001 >/dev/null 2>&1 &

New (autodetect ip)
nohup ./Print2Go -ip auto >/dev/null 2>&1 &

remove non-standard line endings from Print2Go-srv, if any was added.
in openwrt: 
sed -i 's/\r//' /etc/init.d/Print2Go-srv

Edit Print2Go-srv, put the right ip (optional)
vim /etc/init.d/Print2Go-srv

Escape and Save with: :wq 
 
Enable service
/etc/init.d/Print2Go-srv enable

Check enabled
/etc/init.d/Print2Go-srv enabled && echo "on"

Start manually the service to check everything is correct
service Print2Go-srv start

Check status 
service Print2Go-srv status

OpenWRT Luci interface
http://192.168.8.1:81/cgi-bin/luci/