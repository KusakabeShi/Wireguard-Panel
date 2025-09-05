import React, { useState, useMemo, useEffect } from 'react';
import { 
  Box, 
  Select, 
  MenuItem, 
  FormControl, 
  InputLabel,
  IconButton,
  Typography,
  Pagination
} from '@mui/material';
import { 
  Add as AddIcon,
  KeyboardArrowUp as ArrowUpIcon,
  KeyboardArrowDown as ArrowDownIcon
} from '@mui/icons-material';
import ClientItem from './ClientItem';
import stateManager from '../../utils/stateManager';
import { sortClients, getSortDisplayName, getCurrentSort } from '../../utils/sortUtils';

const DEFAULT_CLIENTS_PER_PAGE = 5;

const ClientList = ({ 
  clients,
  clientsState,
  previousClientsState,
  lastUpdateTime,
  previousUpdateTime,
  trafficDisplayMode,
  onTrafficModeToggle,
  expandedClients,
  onToggleExpanded,
  onEdit,
  onDelete,
  onToggle,
  interfaceId,
  serverId,
  onAddClient,
  interfaceInfo,
  serverInfo
}) => {
  const [sortOrder, setSortOrder] = useState(['name-a', 'lastHandshake-d', 'totalTraffic-d', 'enabled-a']);
  const [currentPage, setCurrentPage] = useState(1);
  const [stateInitialized, setStateInitialized] = useState(false);
  const [clientsPerPage, setClientsPerPage] = useState(DEFAULT_CLIENTS_PER_PAGE);

  // Listen for stateManager initialization
  useEffect(() => {
    const syncState = () => {
      if (!stateInitialized) {
        setSortOrder(stateManager.getSortOrder());
        setCurrentPage(stateManager.getServerPage(serverId));
        setClientsPerPage(stateManager.getClientsPerPage());
        setStateInitialized(true);
      }
    };

    // Use the event system instead of polling
    stateManager.onInitialized(syncState);
  }, [serverId, stateInitialized]);

  // Update state when serverId changes (after initialization)
  useEffect(() => {
    if (stateInitialized) {
      setSortOrder(stateManager.getSortOrder());
      setCurrentPage(stateManager.getServerPage(serverId));
      setClientsPerPage(stateManager.getClientsPerPage());
    }
  }, [serverId, stateInitialized]);

  // Save page changes to state manager
  useEffect(() => {
    if (stateInitialized) {
      stateManager.setServerPage(serverId, currentPage);
    }
  }, [serverId, currentPage, stateInitialized]);

  const sortedClients = useMemo(() => {
    if (!clients || clients.length === 0) {
      return [];
    }

    return sortClients(clients, clientsState, sortOrder);
  }, [clients, clientsState, sortOrder]);

  const totalPages = Math.ceil(sortedClients.length / clientsPerPage);
  
  // Ensure currentPage is within bounds
  const validCurrentPage = Math.min(Math.max(1, currentPage), totalPages || 1);
  if (validCurrentPage !== currentPage && totalPages > 0) {
    setCurrentPage(validCurrentPage);
  }
  
  const startIndex = (validCurrentPage - 1) * clientsPerPage;
  const endIndex = startIndex + clientsPerPage;
  const currentClients = sortedClients.slice(startIndex, endIndex);

  const handlePageChange = (event, value) => {
    setCurrentPage(value);
  };

  const currentSort = getCurrentSort(sortOrder);

  const handleSortChange = (event) => {
    const selectedMethod = event.target.value;
    let newOrder;
    
    if (stateInitialized) {
      newOrder = stateManager.updateSortOrder(selectedMethod, currentSort.isAscending);
    } else {
      // Fallback: update sort order locally
      newOrder = [
        `${selectedMethod}-${currentSort.isAscending ? 'a' : 'd'}`,
        ...sortOrder.filter(item => !item.startsWith(selectedMethod + '-'))
      ].slice(0, 4);
    }
    
    setSortOrder(newOrder);
    // Only reset page if current page would be out of bounds after sorting
    const newTotalPages = Math.ceil(sortedClients.length / clientsPerPage);
    if (currentPage > newTotalPages && newTotalPages > 0) {
      setCurrentPage(1);
    }
  };

  const handleSortDirectionToggle = () => {
    const newIsAscending = !currentSort.isAscending;
    let newOrder;
    
    if (stateInitialized) {
      newOrder = stateManager.updateSortOrder(currentSort.method, newIsAscending);
    } else {
      // Fallback: update sort order locally
      newOrder = [
        `${currentSort.method}-${newIsAscending ? 'a' : 'd'}`,
        ...sortOrder.filter(item => !item.startsWith(currentSort.method + '-'))
      ].slice(0, 4);
    }
    
    setSortOrder(newOrder);
    // Only reset page if current page would be out of bounds after sorting
    const newTotalPages = Math.ceil(sortedClients.length / clientsPerPage);
    if (currentPage > newTotalPages && newTotalPages > 0) {
      setCurrentPage(1);
    }
  };

  const handlePerPageChange = (event) => {
    const newPerPage = event.target.value;
    
    // Calculate which client index the user is currently viewing
    const currentFirstClientIndex = (currentPage - 1) * clientsPerPage + 1;
    
    // Calculate new page number to show the same client
    let newPage;
    if (newPerPage === 'all') {
      newPage = 1;
      const newClientsPerPage = sortedClients.length || 1;
      setClientsPerPage(newClientsPerPage);
      if (stateInitialized) {
        stateManager.setClientsPerPage(newClientsPerPage);
      }
    } else {
      const perPageNum = parseInt(newPerPage);
      newPage = Math.ceil(currentFirstClientIndex / perPageNum);
      setClientsPerPage(perPageNum);
      if (stateInitialized) {
        stateManager.setClientsPerPage(perPageNum);
      }
    }
    
    setCurrentPage(newPage);
  };

  if (!clients) {
    return null;
  }

  return (
    <Box>
            {/* Header with Sort, Pagination, and Add Button */}
      <Box sx={{ 
        display: 'flex', 
        alignItems: 'center', 
        justifyContent: 'space-between',
        p: 1, 
        borderTop: '1px solid #e0e0e0' 
      }}>
        {/* Sort Controls */}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          {/* Sort Direction Toggle */}
          <IconButton
            onClick={handleSortDirectionToggle}
            size="small"
            sx={{ 
              backgroundColor: 'rgba(0,0,0,0.04)',
              '&:hover': {
                backgroundColor: 'rgba(0,0,0,0.08)',
              }
            }}
            title={`Sort ${currentSort.isAscending ? 'Descending' : 'Ascending'}`}
          >
            {currentSort.isAscending ? <ArrowUpIcon fontSize="small" /> : <ArrowDownIcon fontSize="small" />}
          </IconButton>

          {/* Sort By Dropdown */}
          <FormControl size="small" sx={{ minWidth: 140 }}>
            <InputLabel>Sort By</InputLabel>
            <Select
              value={currentSort.method}
              onChange={handleSortChange}
              label="Sort By"
            >
              <MenuItem value="name">Name</MenuItem>
              <MenuItem value="lastHandshake">Last Handshake</MenuItem>
              <MenuItem value="totalTraffic">Total Traffic</MenuItem>
              <MenuItem value="enabled">Enabled</MenuItem>
            </Select>
          </FormControl>

          {/* Clients Per Page Dropdown */}
          <FormControl size="small" sx={{ minWidth: 80 }}>
            <InputLabel>Show</InputLabel>
            <Select
              value={clientsPerPage === sortedClients.length ? 'all' : clientsPerPage}
              onChange={handlePerPageChange}
              label="Show"
            >
              <MenuItem value={5}>5</MenuItem>
              <MenuItem value={10}>10</MenuItem>
              <MenuItem value={20}>20</MenuItem>
              <MenuItem value={50}>50</MenuItem>
              <MenuItem value={100}>100</MenuItem>
              <MenuItem value="all">All</MenuItem>
            </Select>
          </FormControl>
        </Box>

        {/* Pagination in Center */}
        {totalPages > 1 && (
          <Pagination 
            count={totalPages}
            page={validCurrentPage}
            onChange={handlePageChange}
            color="primary"
            showFirstButton
            showLastButton
            siblingCount={2}
            boundaryCount={0}
          />
        )}

        {/* Add Client Button */}
        <IconButton 
          onClick={onAddClient}
          sx={{ 
            backgroundColor: '#1976d2',
            color: 'white',
            '&:hover': {
              backgroundColor: '#1565c0',
            }
          }}
        >
          <AddIcon />
        </IconButton>
      </Box>
      {/* Client Items */}
      <Box sx={{ p: 2 }}>
        {clients.length === 0 ? (
          <Box sx={{ 
            textAlign: 'center', 
            py: 0, 
            color: 'text.secondary',
            fontStyle: 'italic' 
          }}>
            No clients yet. Click the + button above to add your first client.
          </Box>
        ) : (
          currentClients.map((client) => (
            <ClientItem
              key={client.id}
              client={client}
              clientState={clientsState[client.id] || null}
              previousClientState={previousClientsState[client.id] || null}
              lastUpdateTime={lastUpdateTime}
              previousUpdateTime={previousUpdateTime}
              trafficDisplayMode={trafficDisplayMode}
              onTrafficModeToggle={onTrafficModeToggle}
              expanded={expandedClients.has(`${interfaceId}_${serverId}_${client.id}`)}
              onToggleExpanded={() => onToggleExpanded(interfaceId, serverId, client.id)}
              onEdit={onEdit}
              onDelete={onDelete}
              onToggle={onToggle}
              interfaceId={interfaceId}
              serverId={serverId}
              interfaceInfo={interfaceInfo}
              serverInfo={serverInfo}
            />
          ))
        )}
      </Box>
    </Box>
  );
};

export default ClientList;