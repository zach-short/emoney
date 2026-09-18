import { useCallback, useSyncExternalStore } from "react";
import { playerStore } from "@/lib/utils/playerHelpers";

const noopSubscribe = () => () => {};

const serverSnapshot = () => null;

/**
 * The player id stored for this room. `null` on the server and on the
 * hydrating render, then the real value once mounted on the client. A direct
 * `playerStore.getPlayerIdForRoom` call in render returns the real value
 * synchronously on the client's very first render too -- before React has
 * matched it against the server-rendered HTML -- which is the hydration
 * mismatch this hook exists to avoid.
 */
export function useStoredPlayerId(roomCode: string) {
  const getSnapshot = useCallback(
    () => playerStore.getPlayerIdForRoom(roomCode),
    [roomCode],
  );

  return useSyncExternalStore(noopSubscribe, getSnapshot, serverSnapshot);
}
