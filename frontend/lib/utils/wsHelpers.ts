// The socket scheme has to match the API's, not the build mode. Keying it off
// NODE_ENV meant a dev build pointed at a remote https API always tried ws://
// and could never connect -- and it would have picked wss:// for a production
// build pointed at a plain-http API.
const getWsUrl = (roomCode: string) => {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
  const { protocol: apiProtocol, host: apiHost } = new URL(apiUrl);

  const protocol = apiProtocol === "https:" ? "wss" : "ws";
  const host = process.env.NEXT_PUBLIC_API_URL_NO_PREFIX ?? apiHost;

  return `${protocol}://${host}/v1/ws/room/${roomCode}`;
};

export { getWsUrl };
