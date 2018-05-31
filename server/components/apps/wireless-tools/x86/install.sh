#!/bin/sh
apt-get install iw isc-dhcp-server haveged
KVER=$(uname -r)
cp interfaces /etc/network/interfaces
cp NetworkManager.conf /etc/NetworkManager/NetworkManager.conf
cp ufw /etc/default/ufw
cp sysctl.conf /etc/ufw/sysctl.conf
cp before.rules /etc/ufw/before.rules
cp dhcpd.conf /etc/dhcp/dhcpd.conf
cp isc-dhcp-server /etc/default/isc-dhcp-server
cp hostapd /etc/default/hostapd
mkdir /etc/hostapd
cp hostapd.conf /etc/hostapd/hostapd.conf
install -D hostapd /usr/local/bin/hostapd
install -D hostapd_cli /usr/local/bin/hostapd_cli
install -p -m 644 rtl8812au.ko /lib/modules/$KVER/kernel/drivers/net/wireless/
/sbin/depmod -a $KVER
