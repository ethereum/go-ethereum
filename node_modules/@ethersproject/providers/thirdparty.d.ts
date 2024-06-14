declare module "ws" {
    export interface WebSocker {
        send(): void;
        onopen: () => void;
        onmessage: (messageEvent: { target: any, type: string, data: string }) => void
    }

    export default WebSocket;
}

