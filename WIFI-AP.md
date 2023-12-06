# Create WIFI Access Point using hostapd
* Modify/create `/etc/hostapd/hostapd.conf` with
```
interface=wlan0
driver=nl80211
ssid=BBAI64
hw_mode=g
channel=7
wpa=1
wpa_passphrase=P@ssw0rd1234
wpa_key_mgmt=WPA-PSK
wpa_pairwise=TKIP CCMP
wpa_ptk_rekey=600
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
* Start AP with
```
sudo service hostapd restart
```
* Stop AP with 
```
sudo service hostapd stop
```
