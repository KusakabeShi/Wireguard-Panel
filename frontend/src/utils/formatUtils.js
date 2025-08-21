export const formatBytes = (bytes) => {
  if (!bytes || bytes === 0) return '0 B';
  
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
};

export const formatLastHandshake = (lastHandshake) => {
  if (!lastHandshake) return 'Never';
  
  const now = new Date();
  const handshakeTime = new Date(lastHandshake);
  const diffMs = now - handshakeTime;
  const diffMins = Math.floor(diffMs / 60000);
  
  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins} min ago`;
  
  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours} hr ago`;
  
  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays} days ago`;
};

export const isClientActive = (lastHandshake) => {
  if (!lastHandshake) return false;
  
  const now = new Date();
  const handshakeTime = new Date(lastHandshake);
  const diffMs = now - handshakeTime;
  const diffMins = Math.floor(diffMs / 60000);
  
  return diffMins < 2; // Active if handshake within 2 minutes
};