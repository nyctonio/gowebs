import Redis from "ioredis";
const subscriber = new Redis({
  port: 6379, // Redis port
  host: "143.198.147.232",
  username: "nio",
  password: "expo_nio23",
});
// redis pub sub channel
const channelName = "channel";
let connected_clients = 0;

const createSubServer = async () => {
  try {
    const server = Bun.serve({
      port: 8080,
      fetch(req, server) {
        // upgrade the request to a WebSocket
        if (server.upgrade(req)) {
          connected_clients++;
          return; // do not return a Response
        }
        console.log("upgrade failed");
        return new Response("Upgrade failed :(", { status: 500 });
      },
      websocket: {
        message(ws, message) {
          ws.send(message); // echo the message back
        }, // a message is received
        open(ws) {
          ws.subscribe(channelName);
        }, // a socket is opened
        close(ws, code, message) {
          connected_clients--;
        }, // a socket is closed
        drain(ws) {}, // the socket is ready to receive more data
        perMessageDeflate: true,
      },
    });
    subscriber.subscribe(channelName, (err, count) => {
      console.log("subscribe", count);
    });
    subscriber.on("message", (channel, message) => {
      console.log("publishing to ",new Date(), connected_clients);
      server.publish(channelName, message);
    });
  } catch (e) {
    console.log("Error", e);
  }
};
createSubServer();
