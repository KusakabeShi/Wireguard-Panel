// Multi-level sorting utility for clients
export const sortClients = (clients, clientsState, sortOrder) => {
  if (!clients || clients.length === 0) {
    return [];
  }

  const sorted = [...clients].sort((a, b) => {
    // Apply each sort method in order until we get a non-zero result
    for (const sortMethod of sortOrder) {
      const [method, direction] = sortMethod.split('-');
      const isAscending = direction === 'a';
      let result = 0;

      switch (method) {
        case 'name':
          result = a.name.localeCompare(b.name);
          break;

        case 'lastHandshake': {
          const aState = clientsState[a.id];
          const bState = clientsState[b.id];
          const aTime = aState?.latestHandshake ? new Date(aState.latestHandshake).getTime() : 0;
          const bTime = bState?.latestHandshake ? new Date(bState.latestHandshake).getTime() : 0;
          result = aTime - bTime; // Natural ascending: oldest first
          break;
        }

        case 'totalTraffic': {
          const aState = clientsState[a.id];
          const bState = clientsState[b.id];
          const aTraffic = (aState?.transferRx || 0) + (aState?.transferTx || 0);
          const bTraffic = (bState?.transferRx || 0) + (bState?.transferTx || 0);
          result = aTraffic - bTraffic; // Natural ascending: lowest first
          break;
        }

        case 'enabled':
          result = (a.enabled ? 1 : 0) - (b.enabled ? 1 : 0); // Natural ascending: disabled first
          break;

        default:
          continue;
      }

      // Apply direction
      if (!isAscending) {
        result = -result;
      }

      // If we have a definitive result, return it
      if (result !== 0) {
        return result;
      }
    }

    // If all sort methods result in equality, maintain stable sort
    return 0;
  });

  return sorted;
};

// Get display name for sort methods
export const getSortDisplayName = (method) => {
  const displayNames = {
    name: 'Name',
    lastHandshake: 'Last Handshake',
    totalTraffic: 'Total Traffic',
    enabled: 'Enabled'
  };
  return displayNames[method] || method;
};

// Get the current primary sort method and direction
export const getCurrentSort = (sortOrder) => {
  if (!sortOrder || sortOrder.length === 0) {
    return { method: 'name', isAscending: true };
  }
  
  const [method, direction] = sortOrder[0].split('-');
  return { method, isAscending: direction === 'a' };
};