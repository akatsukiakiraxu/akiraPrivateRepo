if [ $# -ne 2 ]; then
  echo "sudo ./install.sh [path to gui] [path to api server]"
  exit 1
fi

apt install isc-dhcp-server
ln -sf $1 /opt/olive_gui
ln -sf $2 /opt/olive_apiServer
cp hostapd.service olive-gui.service olive-api-server.service /etc/systemd/system/
cp hostapd olive-api-server olive-gui /opt/
systemctl enable systemd-networkd
systemctl enable systemd-networkd-wait-online
systemctl enable hostapd.service
systemctl enable olive-gui.service
systemctl enable olive-api-server.service
