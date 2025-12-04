const canvas = document.getElementById('gameCanvas');
const ctx = canvas.getContext('2d');
window.gameState = { entities: [], me: { elixir: 5, hand: [], next: "" }, time: 0, myTeam: 0, playerCount: 0 };

const ASSET_PATH = '/static/assets/';
const sprites = {};
const ALL_UNITS = ['morphilina', 'dangerlyoha', 'yuuechka', 'morphe', 'classic_morphe', 'classic_yuu', 'sasavot', 'murzik', 'king_tower', 'princess_tower'];

// VIEW SETTINGS
let SCALE = 32;
let BOARD_OFFSET_Y = 0;
let selectedCard = null;

function getVisualCoords(x, y) {
    const myTeam = window.gameState.myTeam || 0;
    // If I am Team 1 (Top), flip the board so I appear at the bottom
    if (myTeam === 1) {
        return { x: 18 - x, y: 32 - y };
    }
    return { x, y };
}

function loadAssets(onComplete) {
    let loaded = 0;
    if (ALL_UNITS.length === 0) { onComplete(); return; }
    const markLoaded = () => { loaded++; if (loaded === ALL_UNITS.length) onComplete(); };

    ALL_UNITS.forEach(key => {
        const attempts = [`${ASSET_PATH}${key}.png`, `${ASSET_PATH}${key}.PNG`];
        let attempt = 0;
        const img = new Image();

        const tryNext = () => {
            if (attempt >= attempts.length) {
                markLoaded();
                return;
            }
            img.src = attempts[attempt++];
        };

        img.onload = () => { sprites[key] = img; markLoaded(); };
        img.onerror = tryNext;

        tryNext();
    });
}

const handDiv = document.getElementById('hand');
const nextCardDiv = document.getElementById('next-card-container');
const elixirBar = document.getElementById('elixir-bar');
const elixirText = document.getElementById('elixir-text');
const timer = document.getElementById('timer');
const gameOverScreen = document.getElementById('game-over-screen');
const gameOverTitle = document.getElementById('game-over-title');
const playAgainBtn = document.getElementById('play-again');
const lobbyLinks = document.querySelectorAll('[data-lobby-link]');
const waitingScreen = document.getElementById('waiting-screen');
const inviteLink = document.getElementById('invite-link');
const copyInvite = document.getElementById('copy-invite');
const medalDelta = document.getElementById('medal-delta');

if (lobbyLinks && window.netParams) {
    // We only pass the language. We DO NOT pass userID. 
    // This forces the Lobby to look at your Cookies to decide who you are.
    const { lang } = window.netParams; 
    const href = `/?lang=${encodeURIComponent(lang)}`;
    
    lobbyLinks.forEach((link) => {
        link.setAttribute('href', href);
    });
}
if (playAgainBtn) {
    playAgainBtn.addEventListener('click', () => {
        if (window.net && window.net.sendReset) {
            window.net.sendReset();
        }
        gameOverScreen.style.display = 'none';
        // Re-hide the top lobby button when playing again
        const topBackBtn = document.getElementById('hud-back');
        if (topBackBtn) topBackBtn.style.display = 'none'; 
    });
}

function updateUI() {
    if (!handDiv || !nextCardDiv || !elixirBar || !elixirText || !timer) {
        return;
    }

    const me = window.gameState.me;
    if (!me) return;
    const myTeam = window.gameState.myTeam || 0;

    // ELIXIR
    const pct = (me.elixir / 10) * 100;
    elixirBar.style.width = `${pct}%`;
    elixirText.innerText = Math.floor(me.elixir);

    // TIMER
    const t = window.gameState.time;
    let text = "0:00";
    if (window.gameState.tiebreaker) {
        text = "SUDDEN DEATH";
        timer.style.color = "red";
    } else if (window.gameState.overtime) {
        let otSeconds = (120 + 90) - Math.floor(t);
        if (otSeconds < 0) otSeconds = 0;
        text = `OT ${Math.floor(otSeconds/60)}:${(otSeconds%60).toString().padStart(2,'0')}`;
        timer.style.color = "orange";
    } else {
        let sec = 120 - Math.floor(t);
        if (sec < 0) sec = 0;
        text = `${Math.floor(sec/60)}:${(sec%60).toString().padStart(2,'0')}`;
        timer.style.color = "white";
    }
    timer.innerText = text;

    // HAND
    me.hand.forEach((key, i) => {
        let cardDiv = handDiv.children[i];
        if (!cardDiv) {
            cardDiv = document.createElement('div');
            cardDiv.className = 'card';
            handDiv.appendChild(cardDiv);
            
            // --- REMOVED onclick FROM HERE --- 
            // Do not assign onclick inside this if-block!
        }
        
        // --- NEW LOCATION: UPDATE ONCLICK EVERY TIME ---
        // This ensures the click handler always knows the CURRENT card key (key)
        cardDiv.onclick = () => {
            if (selectedCard === key) selectedCard = null;
            else selectedCard = key;
            updateUI();
        };

        if (cardDiv.dataset.cardKey !== key) {
            cardDiv.innerHTML = '';
            const img = sprites[key];
            if (img) {
                const icon = new Image(); icon.src = img.src; icon.style.width = "40px";
                cardDiv.appendChild(icon);
            } else {
                cardDiv.innerText = key.substring(0, 3);
            }
            cardDiv.dataset.cardKey = key;
        }

        if (selectedCard === key) {
            cardDiv.classList.add('selected');
        } else {
            cardDiv.classList.remove('selected');
        }
    });

    // NEXT
    const nextKey = me.next;
    if (nextCardDiv.dataset.cardKey !== nextKey) {
        nextCardDiv.innerHTML = '';
        if (nextKey) {
            if (sprites[nextKey]) {
                 const icon = new Image(); icon.src = sprites[nextKey].src; icon.style.width = "30px";
                 const miniCard = document.createElement('div');
                 miniCard.className = 'card mini';
                 miniCard.appendChild(icon);
                 nextCardDiv.appendChild(miniCard);
            } else { 
                const miniCard = document.createElement('div');
                miniCard.className = 'card mini';
                miniCard.innerText = nextKey.substring(0,3);
                nextCardDiv.appendChild(miniCard);
            }
        }
        nextCardDiv.dataset.cardKey = nextKey;
    }


    // WAITING
    if (waitingScreen) {
        if ((window.gameState.playerCount || 0) < 2 && !window.gameState.gameOver) {
            waitingScreen.style.display = 'flex';
        } else {
            waitingScreen.style.display = 'none';
        }
    }

    // GAME OVER
    if (window.gameState.gameOver) {
        gameOverScreen.style.display = 'flex';

        const topBackBtn = document.getElementById('hud-back');
        if (topBackBtn) topBackBtn.style.display = 'flex';
        const win = window.gameState.winner === myTeam;
        gameOverTitle.innerText = win ? "VICTORY!" : "DEFEAT";
        gameOverTitle.style.color = win ? "#4f4" : "#f44";
        if (medalDelta) {
            medalDelta.textContent = win ? "+30 medals" : "-15 medals";
            medalDelta.style.color = win ? "#4f4" : "#f99";
        }
        const backBtn = document.getElementById('back-to-lobby');
        if (backBtn && window.SERVER_USER_ID) {
            // Force return to profile with ID
            backBtn.href = `/?userID=${window.SERVER_USER_ID}&lang=${window.SERVER_LANG}`;
        }
    } else {
        gameOverScreen.style.display = 'none';
        const topBackBtn = document.getElementById('hud-back');
        if (topBackBtn) topBackBtn.style.display = 'none';
    }
}

window.onGameStateUpdate = updateUI;

canvas.addEventListener('mousedown', (e) => {
    if (!selectedCard) return;
    const rect = canvas.getBoundingClientRect();
    
    // 1. Get Click Coordinates relative to screen
    let rawX = (e.clientX - rect.left) / SCALE;
    let rawY = ((e.clientY - rect.top) - BOARD_OFFSET_Y) / SCALE;

    // 2. UN-FLIP coordinates to send correct Server X/Y
    const myTeam = window.gameState.myTeam || 0;
    let serverX = rawX;
    let serverY = rawY;

    if (myTeam === 1) {
        serverX = 18 - rawX;
        serverY = 32 - rawY;
    }

    // 3. Clamp
    if (serverY < 0) serverY = 0;
    if (serverY > 32) serverY = 32;

    if (window.net && window.net.sendSpawn) {
        // Send SERVER coordinates
        window.net.sendSpawn(selectedCard, serverX, serverY);
        selectedCard = null;
        updateUI();
    }
});

function render() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.save();
    ctx.translate(0, BOARD_OFFSET_Y);

    // --- MAP DRAWING ---
    // We draw the map static, because we flip entities on top of it.
    // However, we want the "Enemy Side" tint to always be at the top visually.
    
    ctx.fillStyle = "rgba(255, 0, 0, 0.1)"; 
    ctx.fillRect(0, 0, 18 * SCALE, 16 * SCALE); // Top half is enemy

    // River (Center)
    ctx.fillStyle = "#4da6ff"; 
    ctx.fillRect(0, 15.5 * SCALE, 18 * SCALE, 1 * SCALE);
    
    // Bridges
    ctx.fillStyle = "#8B4513";
    ctx.fillRect(2.5 * SCALE, 14.5 * SCALE, 2 * SCALE, 3 * SCALE);
    ctx.fillRect(13.5 * SCALE, 14.5 * SCALE, 2 * SCALE, 3 * SCALE);

    // Spawn Hint
    if (selectedCard) {
        // Visual cue where you can drop
        ctx.fillStyle = "rgba(255, 255, 255, 0.2)";
        // Logic: You can always spawn in bottom half visually
        ctx.fillRect(0, 16 * SCALE, 18 * SCALE, 16 * SCALE);
        // ... (dashed line code) ...
    }

    const entities = window.gameState.entities || [];
    const myTeam = window.gameState.myTeam || 0;

    // Sort by VISUAL Y so they layer correctly
    const sorted = entities.map(e => {
        const v = getVisualCoords(e.x, e.y);
        return { ...e, visX: v.x, visY: v.y };
    }).sort((a, b) => a.visY - b.visY);

    sorted.forEach(ent => {
        const screenX = ent.visX * SCALE;
        const screenY = ent.visY * SCALE;
        const img = sprites[ent.key];

        ctx.fillStyle = "rgba(0,0,0,0.3)";
        ctx.beginPath(); 
        ctx.ellipse(screenX, screenY, SCALE/3, SCALE/6, 0, 0, Math.PI * 2); 
        ctx.fill();

        if (img) {
            let size = SCALE * 2;
            if (ent.key.includes("tower")) size = SCALE * 3;
            ctx.drawImage(img, screenX - size/2, screenY - size + 5, size, size);
        } else {
            // Fallback Box
            ctx.fillStyle = ent.team === myTeam ? 'blue' : 'red';
            let w=SCALE*0.8, h=SCALE;
            ctx.fillRect(screenX - w/2, screenY - h, w, h);
        }

        // HP Bar
        if (ent.hp < ent.max_hp) {
             const hpPct = ent.hp / ent.max_hp;
             const barW = SCALE;
             // Color logic: My team is always Green, Enemy always Red
             const barColor = (ent.team === myTeam) ? '#4f4' : '#f44';
             
             ctx.fillStyle = 'black'; 
             ctx.fillRect(screenX - barW/2, screenY - SCALE*1.2, barW, 5);
             ctx.fillStyle = barColor; 
             ctx.fillRect(screenX - barW/2, screenY - SCALE*1.2, barW * hpPct, 5);
        }
    });

    ctx.restore();
    requestAnimationFrame(render);
}
/*
function render() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    ctx.save();
    ctx.translate(0, BOARD_OFFSET_Y);

    // MAP
    ctx.fillStyle = "rgba(255, 0, 0, 0.1)"; // Enemy Side Tint
    ctx.fillRect(0, 0, 18 * SCALE, 16 * SCALE);

    // River
    ctx.fillStyle = "#4da6ff"; ctx.fillRect(0, 15.5 * SCALE, 18 * SCALE, 1 * SCALE);
    // Bridges
    ctx.fillStyle = "#8B4513";
    ctx.fillRect(2.5 * SCALE, 14.5 * SCALE, 2 * SCALE, 3 * SCALE);
    ctx.fillRect(13.5 * SCALE, 14.5 * SCALE, 2 * SCALE, 3 * SCALE);

    // Spawn Hint
    if (selectedCard) {
        ctx.fillStyle = "rgba(255, 255, 255, 0.2)";
        ctx.fillRect(0, 16 * SCALE, 18 * SCALE, 16 * SCALE);
        ctx.strokeStyle = "rgba(255, 255, 255, 0.8)";
        ctx.lineWidth = 2;
        ctx.setLineDash([10, 10]);
        ctx.strokeRect(0, 16 * SCALE, 18 * SCALE, 16 * SCALE);
        ctx.setLineDash([]);
    }

    const entities = window.gameState.entities || [];
    const sorted = entities.sort((a, b) => a.y - b.y);

    sorted.forEach(ent => {
        const screenX = ent.x * SCALE;
        const screenY = ent.y * SCALE;
        const img = sprites[ent.key];

        ctx.fillStyle = "rgba(0,0,0,0.3)";
        ctx.beginPath(); ctx.ellipse(screenX, screenY, SCALE/3, SCALE/6, 0, 0, Math.PI * 2); ctx.fill();

        if (img) {
            let size = SCALE * 2;
            if (ent.key.includes("tower")) size = SCALE * 3;
            ctx.drawImage(img, screenX - size/2, screenY - size + 5, size, size);
        } else {
            ctx.fillStyle = ent.team === 0 ? 'blue' : 'red';
            let w=SCALE*0.8, h=SCALE;
            if (ent.key.includes("tower")) { w=SCALE*1.5; h=SCALE*2; }
            ctx.fillRect(screenX - w/2, screenY - h, w, h);
        }

        if (ent.hp < ent.max_hp) {
             const hpPct = ent.hp / ent.max_hp;
             const barW = SCALE;
             ctx.fillStyle = 'black'; ctx.fillRect(screenX - barW/2, screenY - SCALE*1.2, barW, 5);
             ctx.fillStyle = ent.team === 0 ? '#4f4' : '#f44'; ctx.fillRect(screenX - barW/2, screenY - SCALE*1.2, barW * hpPct, 5);
        }
    });

    ctx.restore();
    requestAnimationFrame(render);
}
    */

function resizeCanvas() {
    let h = window.innerHeight;
    let w = h * (9/16);
    if (w > window.innerWidth) { w = window.innerWidth; h = w / (9/16); }
    canvas.width = w; canvas.height = h;
    SCALE = w / 18;
    BOARD_OFFSET_Y = 0;
}

window.addEventListener('resize', resizeCanvas);
window.onload = () => {
    resizeCanvas();
    const url = new URL(window.location.href);
    url.searchParams.delete('userID'); // Remove ID so friend doesn't become you
    const link = url.toString();
    
    if (inviteLink) inviteLink.value = link;
    loadAssets(() => { render(); });
};

if (copyInvite && inviteLink) {
    copyInvite.onclick = async () => {
        try {
            await navigator.clipboard.writeText(inviteLink.value);
            copyInvite.textContent = 'Copied';
            setTimeout(() => copyInvite.textContent = 'Copy', 1200);
        } catch (e) {
            copyInvite.textContent = 'Failed';
        }
    };
}
