const urlParams = new URLSearchParams(window.location.search);
const getCookie = (name) => {
    const m = document.cookie.match(new RegExp('(?:^|; )' + name.replace(/([$?*|{}()[\]\\/+^])/g, '\\$1') + '=([^;]*)'));
    return m ? decodeURIComponent(m[1]) : '';
};
const cookieUser = getCookie('user_id');
const userID = urlParams.get('userID') || cookieUser || 'guest';
const lang = urlParams.get('lang') || 'en';
window.netParams = { userID, lang };
document.documentElement.lang = lang;

const protocol = window.location.protocol === "https:" ? "wss" : "ws";
const socket = new WebSocket(`${protocol}://${window.location.host}/ws?userID=${userID}`);

socket.onopen = () => console.log("Connected with userID:", userID);
socket.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    if (msg.type === "state") {
        if (window.gameState) {
            window.gameState.entities = msg.entities;
            window.gameState.time = msg.time;
            window.gameState.gameOver = msg.gameOver;
            window.gameState.winner = msg.winner;
            window.gameState.overtime = msg.overtime;
            window.gameState.tiebreaker = msg.tiebreaker;
            window.gameState.playerCount = msg.playerCount || 0;
            if (msg.me) {
                window.gameState.me = msg.me;
                window.gameState.myTeam = msg.myTeam;
            }
        }
        if (window.onGameStateUpdate) {
            window.onGameStateUpdate();
        }
    }
};
window.net = {
    sendSpawn: (cardKey, x, y) => {
        if (socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ type: "spawn", key: cardKey, x: x, y: y }));
    },
    sendReset: () => {
        if (socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ type: "reset" }));
    }
};
