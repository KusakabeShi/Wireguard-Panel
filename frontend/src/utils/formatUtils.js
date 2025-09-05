export const formatBytes = (bytes) => {
  if (!bytes || bytes === 0) return '0 B';
  
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB',"PB", "EB", "ZB", "YB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

export const formatLastHandshake = (now, lastHandshake) => {
  if (!lastHandshake) return 'Never';

  const handshakeTime = new Date(lastHandshake);
  const diffMs = now - handshakeTime;
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 3) return 'Just now';
  if (diffSec < 60) return `${diffSec} sec ago`;

  const diffMin = Math.floor(diffSec / 60);
  const sec = diffSec % 60;
  if (diffMin < 60) return `${diffMin} min ${sec} sec ago`;

  const diffHr = Math.floor(diffMin / 60);
  const min = diffMin % 60;
  if (diffHr < 24) return `${diffHr} hr ${min} min ago`;

  const diffDay = Math.floor(diffHr / 24);
  const hr = diffHr % 24;
  return `${diffDay} day${diffDay > 1 ? 's' : ''} ${hr} hr ago`;
};

export const isClientActive = (now,lastHandshake) => {
  if (!lastHandshake) return false;
  
  const handshakeTime = new Date(lastHandshake);
  const diffMs = now - handshakeTime;
  const diffSecs = Math.floor(diffMs / 1000);
  
  return diffSecs <= 121; // Active if handshake within 2 minutes
};

export const formatTransferRate = (bytesPerSecond) => {
  if (!bytesPerSecond || bytesPerSecond === 0) return '0 Bps';
  let bitsPerSecond = bytesPerSecond * 8;
  const k = 1024;
  const sizes = ['Bps', 'Kbps', 'Mbps', 'Gbps', 'Tbps',"Pbps", "Ebps", "Zbps", "Ybps"];
  const i = Math.floor(Math.log(Math.abs(bitsPerSecond)) / Math.log(k));
  
  const rate = (bitsPerSecond / Math.pow(k, i));
  const formattedRate = rate < 10 ? rate.toFixed(2) : rate < 100 ? rate.toFixed(1) : Math.round(rate);
  
  return `${formattedRate} ${sizes[i]}`;
};

export const calculateTransferRate = (currentBytes, previousBytes, currentTime, previousTime) => {
  if (!currentBytes || !previousBytes || !currentTime || !previousTime) {
    return 0;
  }
  
  const bytesDiff = currentBytes - previousBytes;
  const timeDiff = (currentTime - previousTime) / 1000; // Convert to seconds
  if (timeDiff <= 0) return 0;
  
  return bytesDiff / timeDiff;
};

// Traffic display mode constants and utilities
export const TRAFFIC_DISPLAY_MODES = {
  TOTAL: 'total',
  RATE: 'rate'
};

export const getTrafficDisplayMode = () => {
  // Import stateManager locally to avoid circular dependency
  const stateManager = require('./stateManager').default;
  const mode = stateManager.getTrafficDisplayMode();
  return mode && Object.values(TRAFFIC_DISPLAY_MODES).includes(mode) 
    ? mode 
    : TRAFFIC_DISPLAY_MODES.TOTAL;
};

export const setTrafficDisplayMode = (mode) => {
  if (Object.values(TRAFFIC_DISPLAY_MODES).includes(mode)) {
    // Import stateManager locally to avoid circular dependency
    const stateManager = require('./stateManager').default;
    stateManager.setTrafficDisplayMode(mode);
  }
};