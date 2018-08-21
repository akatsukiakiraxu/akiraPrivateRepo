#/bin/sh
file="./app.properties"
if [ -f "$file" ]
then
  echo "$file found."
  while IFS='=' read -r key value
  do
    eval ${key}=\${value}
  done < "$file"
  echo "dev="${dev}
  echo "ip="${ip}
  ip addr flush dev ${dev}
  ip addr add ${ip}/${netmask} ${gateway} dev ${dev}
  ip link set ${dev} up
  systemctl restart smbd nmbd
else
  echo "$file not found."
fi
