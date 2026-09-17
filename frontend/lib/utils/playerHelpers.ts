export interface PlayerGame {
  roomCode: string;
  playerId: string;
}

// These run during the server render too -- `/room/[code]` reads the stored id
// while rendering. Touching localStorage there threw, so a direct hit on a room
// URL (a refresh, a bookmark, a shared link) returned a 500; the room only ever
// worked when reached by client-side navigation from join/create.
const hasStorage = () => typeof window !== "undefined" && !!window.localStorage;

export const playerStore = {
  setPlayerIdForRoom(roomCode: string, playerId: string) {
    if (!hasStorage()) return;
    localStorage.setItem(`room_${roomCode}_playerId`, playerId);
  },

  getPlayerIdForRoom(roomCode: string): string | null {
    if (!hasStorage()) return null;
    return localStorage.getItem(`room_${roomCode}_playerId`);
  },

  clearPlayerDataForRoom(roomCode: string) {
    if (!hasStorage()) return;
    localStorage.removeItem(`room_${roomCode}_playerId`);
  },

  clearAllPlayerData() {
    if (!hasStorage()) return;
    Object.keys(localStorage).forEach((key) => {
      if (key.startsWith("room_") && key.endsWith("_playerId")) {
        localStorage.removeItem(key);
      }
    });
  },
};
