import {
  createConnectTransport,
  createPromiseClient,
} from "@bufbuild/connect-web";
import { Time } from "../pb/time_connect";
import { Empty } from "@bufbuild/protobuf";

const transport = createConnectTransport({
  baseUrl: "/api",
  credentials: "include",
  useBinaryFormat: true,
});

const client = createPromiseClient(Time, transport);

async function stream(tag: number) {
  for await (const msg of client.live(new Empty(), {
    headers: {
      "X-Request-ID": Date.now() + "",
    },
  })) {
    console.log(`stream#${tag}`, msg.toDate());
  }
}

async function main() {
  for (let i = 0; i < 50; i++) {
    stream(i + 1);
  }
}

main().then(console.log).catch(console.error);
