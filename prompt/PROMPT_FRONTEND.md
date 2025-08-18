I'm going to implement a wireguard server panel:
This is an example OpenAPI v3 for this backend:
```
read API_SPEC.yaml
```
Write the frontend for this project:
Programming language: React(frontend) with Material UI
Top: Title with setting button
Left: Vertical Tabs. List all interfaces, show interfaces name and a "+" which shows a "new interfaces dialog".

In the interfaces Tab:
    First row: Interface row. Show interface name:port and a gear icon, which shows a "edit interface dialog"
    Remain rows Servers belongs to this interface: 
        Shows server name, Server Netowrks(v4 and v6) a "enable/disable switch" to tnable/disable this server, a "edit button", shows a "edit server dialog". A Expand button
        Expandable: Contains Client list of this server
    Last row: Shows a "+", which opens a "new server dialog"
Client row: Show name, up/down traffic, status icon(if lasthandshake smaller than 2 min), enable/disable switch, a "edit button", shows a "edit client dialog". and a expand button.
    Expandable: It shows IP, IPv6, Public key,lasthandshake. endpoint(IP:port read from wg status) and a copy button with wireguard config textarea.
Dialog: New/Edit uses same dialog with different title. And "New" dialog hides the "delete" button

Create the webpage depicted in the provided screenshot, don't contain the browser frame.
The replica should maintain the exact visual layout, including the positioning of all elements, typography, and color scheme.
Ensure all interactive components function as they do in the original design.
```
read wg_panrl.png
```