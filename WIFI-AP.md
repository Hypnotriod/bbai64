# Create WIFI Access Point using hostapd
* Modify/create `/etc/hostapd/hostapd.conf` with
```
interface=wlan0
driver=nl80211
ssid=BBAI64
hw_mode=g
channel=7
ap_max_inactivity=3600
wpa=1
wpa_passphrase=P@ssw0rd1234
wpa_key_mgmt=WPA-PSK
wpa_pairwise=TKIP CCMP
wpa_ptk_rekey=3600
macaddr_acl=0
```
* Modify/create `/lib/systemd/network/hostapd.network` with
```
[Match]
WLANInterfaceType=ap
Name=wlan0

[Network]
Address=192.168.50.1/24
DHCPServer=yes
IPMasquerade=yes
```
* Unmask and disable `hostapd`
```
sudo systemctl unmask hostapd
sudo systemctl disable --now hostapd
```
* Start AP with
```
sudo service hostapd restart
```
* Stop AP with 
```
sudo service hostapd stop
```
* Start AP on startup  
Add `disabled=1` to `/etc/wpa_supplicant/wpa_supplicant-wlan0.conf`
```
disabled=1
```
Add to `/etc/rc.local`
```
# start wi-fi ap
service hostapd restart
```