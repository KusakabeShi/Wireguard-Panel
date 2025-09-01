# WireGuard Panel Frontend Implementation

## Overview
Implement a React-based frontend for the WireGuard server management panel.

## Backend Integration
This frontend connects to a REST API backend. The API specification is available at:
```
@prompt/API_SPEC.yaml
```

## Technical Requirements
- **Framework**: React with Create React App (CRA)
- **UI Library**: Material UI
- **Build Output**: `frontend/build/` directory
- **Proxy**: Use CRA proxy to connect to backend (configured via `config.json`)

## Layout Structure:
**Header**: Blue header bar with "WG-Panel" title and settings gear icon on the right

**Sidebar (Left Panel)**: 
- Red-bordered vertical interface list with "Interfaces" header and hamburger menu icon
- Interface items (if1, if2) displayed as tabs/buttons
- "+" button at bottom for adding new interfaces
- When no interface selected: shows empty state with "+" button in main area

**Main Content Area**:
When interface selected (if1):
- **Interface Header**: Shows interface name and endpoint (e.g., "if1 if1.url.com:51200") with gear icon for editing
- **Server Rows**: Each server shows:
  - Server name and network (e.g., "server1 192.168.21/24") 
  - Toggle switches for enable/disable
  - Edit button for server edit
  - Expand arrow action buttons
- **Client Rows** (when server expanded): Shows:
  - Client name with traffic stats (e.g., "client 1 ↑ 16.8MB ↓ 33.7MB")
  - Status indicator (green if handshake < 2 min)
  - Toggle switch for enable/disableand 
  - Edit button for client edit
  - Expand arrow for details
- **Client Details** (when client expanded): Table showing:
  - IP: "null" or IPv4 address
  - IPv6: "null" or IPv6 address  
  - Transferred: Traffic statistics
  - Last handshake: Time since last communication
  - Endpoint: Connection endpoint
  - WireGuard config: Textarea with config and copy button
- **Client Add Buttons**: "+" buttons for adding new client at bottom of each server section
- **Server Add Buttons**: "+" buttons for adding new servers after the last server section

## Dialog Components:
Three modal dialogs with consistent styling (white background, form fields, action buttons):

### 1. New/Edit Interface Dialog
Simple form layout with the following fields:
- Name (text input)
- Endpoint (text input)  
- Port (text input)
- MTU (text input)
- Private Key (text input)
- VRF Name (text input)
- FWMark (text input)

### 2. New/Edit Server Dialog
**Complex hierarchical form with dependency-based enabling/disabling:**

**Top Level:**
- Name (text input)
- DNS (text input)

**IPv4 Section:**
- ☑️ IPv4 (checkbox) - **Master control for entire IPv4 hierarchy**
  - When **unchecked**: All IPv4 sub-sections and fields become disabled/grayed out
  - When **checked**: IPv4 sub-sections become available

**IPv4 Sub-sections (enabled only when IPv4 checkbox is checked):**
- **Network** (text input)
- **Pseudo-bridge master interface** (text input)
- **Routed Networks** (textarea)
- ☑️ **Routed Networks Firewall** (checkbox)
- ☑️ **SNAT** (checkbox) - **Controls SNAT sub-hierarchy**
  - When **unchecked**: SNAT sub-fields become disabled
  - When **checked**: SNAT sub-fields become available

**SNAT Sub-fields (enabled only when SNAT checkbox is checked):**
- **SNAT IP** (text input)
- **SNAT Excluded Network** (text input)  
- **SNAT Roaming master interface** (text input)

**IPv6 Section:**
- ☑️ IPv6 (checkbox) - **Master control for entire IPv6 hierarchy**
  - When **unchecked**: All IPv6 sub-sections and fields become disabled/grayed out
  - When **checked**: IPv6 sub-sections become available

**IPv6 Sub-sections (enabled only when IPv6 checkbox is checked):**
- **Network** (text input)
- **Pseudo-bridge master interface** (text input)
- **Routed Networks** (textarea)
- ☑️ **Routed Networks Firewall** (checkbox)
- ☑️ **SNAT** (checkbox) - **Controls IPv6 SNAT sub-hierarchy**
  - When **unchecked**: IPv6 SNAT sub-fields become disabled
  - When **checked**: IPv6 SNAT sub-fields become available

**IPv6 SNAT Sub-fields (enabled only when IPv6 SNAT checkbox is checked):**
- **SNAT IP/Network** (text input)
- **SNAT Excluded Network** (text input)
- **SNAT Roaming master interface** (text input)  
- ☑️ **SNAT NETMAP pseudo-bridge** (checkbox)

**Validation Rules:**
- At least one of IPv4 or IPv6 must be enabled
- Display warning messages when conflicting options are selected

### 3. New/Edit Client Dialog
Simple form layout with the following fields:
- Name (text input)
- DNS (text input)
- IP (text input)
- IPv6 (text input)
- Private key (text input)
- Public key (text input)
- Preshared Key (text input)
- Keepalive (text input)

**All dialogs have DELETE, CANCEL, SAVE buttons (DELETE hidden for "New" dialogs)**

## Authentication & API Flow
1. **Initial Load**: Fetch interface list from API
2. **Authentication**: If API returns auth error, display login dialog
3. **Data Flow**: Use API endpoints defined in `API_SPEC.yaml`


For detailed functionality of each area and components, refer to this document:
```
@prompt/PROMPT.md
```

## Design Requirements
The application must exactly match the visual design shown in the reference screenshot:
```
@prompt/wg_panel.png
```

**Critical Design Guidelines:**
- Maintain exact visual layout and element positioning
- Preserve typography and color schemes
- Ensure all interactive components function as shown
- Match the browser-less interface design precisely


Please implement the frontend based on the description above.