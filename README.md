# openwrt-ap-dashboard — Starter Repo

Quickstart:

1. source .envrc (or use direnv)
2. cd backend && go build && ./backend -config ../stations.yaml
3. cd frontend && npm install && npm run dev
4. Open frontend (Vite default http://localhost:5173)


### ubus permissions proc net arp
```
cat <<EOF > /etc/acl_arp.json
{
    "arp_access": {
        "description": "Whitelist fuer ARP Pfad",
        "read": {
            "file": [ "/proc/net/arp" ]
        }
    }
}
EOF
```

```
cat <<EOF > /etc/uci-defaults/99-arp-link
#!/bin/sh
ln -sf /etc/acl_arp.json /usr/share/rpcd/acl.d/arp.json
/etc/init.d/rpcd restart
exit 0
EOF
chmod +x /etc/uci-defaults/99-arp-link
# Jetzt einmalig ausführen:
sh /etc/uci-defaults/99-arp-link
```

```
uci add_list rpcd.@login[0].acl='arp_access'
uci commit rpcd
/etc/init.d/rpcd restart
```