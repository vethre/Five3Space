
const canvas = document.getElementById('fishingCanvas');
const ctx = canvas.getContext('2d');
const scoreEl = document.getElementById('scoreValue');
const actionBtn = document.getElementById('actionBtn');
const toastEl = document.getElementById('toast');
const gameOverModal = document.getElementById('gameOverModal');
const finalScoreVal = document.getElementById('finalScoreVal');
const replayBtn = document.getElementById('replayBtn');
const timerEl = document.getElementById('gameTimer');
const reelingUI = document.getElementById('reelingUI');
const tensionBar = document.getElementById('tensionBar');
const progressBar = document.getElementById('progressBar');
const tensionWarning = document.getElementById('tensionWarning');

// Game State
let gameState = 'IDLE'; // IDLE, CASTING, WAITING, BITING, REELING, CAUGHT, GAMEOVER
let score = 0;
let lastTime = 0;
let fishTimer = 0;
let biteTimer = 0;
let gameDuration = 60; // 1 minute round
let timeLeft = gameDuration;
let gameStarted = false;

// Reeling Physics
let tension = 0;
let catchProgress = 0;
let isReeling = false;
const TENSION_LIMIT = 100;
const REEL_STRENGTH = 40; // Tension increase per second
const TENSION_RECOVERY = 30; // Tension decrease per second
const PROGRESS_SPEED = 25; // Progress per second when reeling
const FISH_STRUGGLE = 15; // Progress lost per second when not reeling

// Assets (Procedural for now)
const colors = {
    skyTop: '#1a2a6c',
    skyBottom: '#fdbb2d',
    ice: '#e3f2fd',
    water: '#0d47a1',
    hole: '#01579b'
};

// Player Position
let playerX = window.innerWidth / 2;
let playerY = window.innerHeight * 0.7;

// Physics
let line = {
    startX: 0,
    startY: 0,
    endX: 0,
    endY: 0,
    targetX: 0,
    targetY: 0,
    state: 'RETRACTED'
};

let float = {
    x: 0,
    y: 0,
    bobOffset: 0,
    submerged: false
};

// Particles (Snow)
const particles = [];
for (let i = 0; i < 150; i++) {
    particles.push({
        x: Math.random() * window.innerWidth,
        y: Math.random() * window.innerHeight,
        vx: (Math.random() - 0.5) * 1,
        vy: Math.random() * 2 + 1,
        size: Math.random() * 3
    });
}

function resize() {
    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;
    playerX = canvas.width / 2;
    playerY = canvas.height * 0.7;
}
window.addEventListener('resize', resize);
resize();

function showToast(msg, duration = 1000) {
    toastEl.innerText = msg;
    toastEl.classList.add('show');
    toastEl.style.opacity = '1';
    setTimeout(() => {
        toastEl.classList.remove('show');
        toastEl.style.opacity = '0';
    }, duration);
}

function formatTime(seconds) {
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
}

function startGame() {
    score = 0;
    timeLeft = gameDuration;
    scoreEl.innerText = score;
    gameState = 'IDLE';
    gameStarted = true;
    updateUI();
    gameOverModal.classList.add('hidden');
    lastTime = performance.now();
    requestAnimationFrame(loop);
}

function updateUI() {
    const t = window.translations;
    if (gameState === 'IDLE') {
        actionBtn.style.display = 'block';
        actionBtn.innerText = t.cast;
        actionBtn.classList.remove('reef');
        actionBtn.disabled = false;
        reelingUI.style.display = 'none';
        actionBtn.style.opacity = 1;
    } else if (gameState === 'WAITING' || gameState === 'CASTING') {
        actionBtn.style.display = 'block';
        actionBtn.innerText = t.wait;
        actionBtn.disabled = true;
        actionBtn.style.opacity = 0.5;
        reelingUI.style.display = 'none';
    } else if (gameState === 'BITING') {
        actionBtn.style.display = 'block';
        actionBtn.innerText = t.reel;
        actionBtn.className = 'action-btn reef';
        actionBtn.disabled = false;
        actionBtn.style.opacity = 1;
        reelingUI.style.display = 'none';
    } else if (gameState === 'REELING') {
        actionBtn.style.display = 'block';
        actionBtn.innerText = t.reel;
        actionBtn.className = 'action-btn reef';
        reelingUI.style.display = 'flex';
    } else {
        actionBtn.style.display = 'none';
        reelingUI.style.display = 'none';
    }
}

// Support both click and hold
actionBtn.addEventListener('mousedown', () => { if (gameState === 'REELING') isReeling = true; });
actionBtn.addEventListener('mouseup', () => { isReeling = false; });
actionBtn.addEventListener('touchstart', (e) => { e.preventDefault(); if (gameState === 'REELING') isReeling = true; });
actionBtn.addEventListener('touchend', () => { isReeling = false; });

actionBtn.addEventListener('click', () => {
    if (gameState === 'IDLE') {
        castLine();
    } else if (gameState === 'BITING') {
        startReeling();
    }
});

function castLine() {
    gameState = 'CASTING';
    float.x = playerX + 60;
    float.y = playerY - 60;
    line.targetX = playerX + 200;
    line.targetY = playerY + 80;
    updateUI();
}

function startReeling() {
    gameState = 'REELING';
    tension = 0;
    catchProgress = 0;
    isReeling = true;
    updateUI();
}

function reelInSuccess() {
    score++;
    scoreEl.innerText = score;
    showToast(window.translations.catch, 1500);
    createWaterSplash(float.x, float.y);
    resetFishing();
}

function reelInFailure(msg) {
    showToast(msg || window.translations.miss, 1500);
    resetFishing();
}

function resetFishing() {
    gameState = 'IDLE';
    float.submerged = false;
    tension = 0;
    catchProgress = 0;
    isReeling = false;
    updateUI();
}

function createWaterSplash(x, y) {
    for (let i = 0; i < 30; i++) {
        particles.push({
            x: x,
            y: y,
            vx: (Math.random() - 0.5) * 15,
            vy: -Math.random() * 15,
            size: Math.random() * 4 + 2,
            life: 1.0,
            type: 'splash'
        });
    }
}

function loop(timestamp) {
    if (!gameStarted) return;
    const dt = (timestamp - lastTime) / 1000;
    lastTime = timestamp;

    update(dt);
    draw();

    if (timeLeft > 0) {
        requestAnimationFrame(loop);
    } else {
        endGame();
    }
}

function update(dt) {
    timeLeft -= dt;
    timerEl.innerText = formatTime(Math.max(0, timeLeft));

    // Snow and Splash particles
    for (let i = particles.length - 1; i >= 0; i--) {
        const p = particles[i];
        if (p.type === 'splash') {
            p.x += p.vx;
            p.y += p.vy;
            p.vy += 0.8; // higher gravity
            p.life -= dt * 2.5;
            if (p.life <= 0) particles.splice(i, 1);
        } else {
            p.x += p.vx;
            p.y += p.vy;
            if (p.y > canvas.height) p.y = -10;
            if (p.x > canvas.width) p.x = 0;
            if (p.x < 0) p.x = canvas.width;
        }
    }

    // Fishing State Machine
    if (gameState === 'CASTING') {
        float.x += (line.targetX - float.x) * 4 * dt;
        float.y += (line.targetY - float.y) * 4 * dt;
        if (Math.abs(float.x - line.targetX) < 5) {
            gameState = 'WAITING';
            fishTimer = Math.random() * 4 + 2;
            updateUI();
        }
    }

    if (gameState === 'WAITING') {
        fishTimer -= dt;
        float.bobOffset = Math.sin(Date.now() / 300) * 3;
        if (fishTimer <= 0) {
            gameState = 'BITING';
            biteTimer = 1.2;
            updateUI();
            float.submerged = true;
            // Immediate splash for bite
            createWaterSplash(float.x, float.y);
        }
    }

    if (gameState === 'BITING') {
        biteTimer -= dt;
        float.bobOffset = Math.sin(Date.now() / 50) * 8;
        if (biteTimer <= 0) {
            reelInFailure(window.translations.miss);
        }
    }

    if (gameState === 'REELING') {
        if (isReeling) {
            tension += REEL_STRENGTH * dt;
            catchProgress += PROGRESS_SPEED * dt;
        } else {
            tension -= TENSION_RECOVERY * dt;
            catchProgress -= FISH_STRUGGLE * dt;
        }

        tension = Math.max(0, tension);
        catchProgress = Math.max(0, catchProgress);

        // Visual tension feedback
        tensionBar.style.width = `${tension}%`;
        progressBar.style.width = `${catchProgress}%`;

        if (tension > 70) {
            tensionWarning.classList.add('visible');
        } else {
            tensionWarning.classList.remove('visible');
        }

        if (tension >= TENSION_LIMIT) {
            reelInFailure("Line Snapped!");
        } else if (catchProgress >= 100) {
            reelInSuccess();
        }
    }
}

function draw() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    // Dynamic Sky
    const grd = ctx.createLinearGradient(0, 0, 0, canvas.height * 0.7);
    grd.addColorStop(0, colors.skyTop);
    grd.addColorStop(1, colors.skyBottom);
    ctx.fillStyle = grd;
    ctx.fillRect(0, 0, canvas.width, canvas.height);

    // Ground / Ice
    ctx.fillStyle = colors.ice;
    ctx.fillRect(0, playerY + 20, canvas.width, canvas.height - playerY);

    // Fishing Hole
    if (gameState !== 'IDLE') {
        ctx.fillStyle = colors.hole;
        ctx.beginPath();
        ctx.ellipse(playerX + 200, playerY + 80, 50, 15, 0, 0, Math.PI * 2);
        ctx.fill();
    }

    drawPlayer(playerX, playerY);

    // Draw Line
    if (gameState !== 'IDLE') {
        ctx.strokeStyle = 'rgba(255,255,255,0.6)';
        ctx.lineWidth = 1.5;
        ctx.beginPath();
        ctx.moveTo(playerX + 60, playerY - 60); // Tip of rod

        let fy = float.y + (gameState === 'WAITING' || gameState === 'BITING' ? float.bobOffset : 0);
        if (float.submerged) fy += 15;

        // Slack curve
        const cpX = (playerX + 60 + float.x) / 2;
        const cpY = Math.max(playerY - 60, fy) + 40;
        ctx.quadraticCurveTo(cpX, cpY, float.x, fy);
        ctx.stroke();

        // Bobber
        ctx.fillStyle = 'white';
        ctx.beginPath();
        ctx.arc(float.x, fy, 6, 0, Math.PI * 2);
        ctx.fill();
        ctx.fillStyle = 'red';
        ctx.beginPath();
        ctx.arc(float.x, fy, 6, 0, Math.PI, true);
        ctx.fill();
    }

    // Snow and particles
    particles.forEach(p => {
        ctx.fillStyle = p.type === 'splash' ? '#4facfe' : 'white';
        ctx.globalAlpha = p.life !== undefined ? p.life : 0.8;
        ctx.beginPath();
        ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2);
        ctx.fill();
    });
    ctx.globalAlpha = 1.0;
}

function drawPlayer(x, y) {
    // Suit
    ctx.fillStyle = '#ff7043';
    ctx.beginPath();
    ctx.roundRect(x - 20, y - 10, 40, 50, 10);
    ctx.fill();

    // Head / Mask
    ctx.fillStyle = '#ffcc80';
    ctx.beginPath();
    ctx.arc(x, y - 35, 18, 0, Math.PI * 2);
    ctx.fill();

    // Cold-weather hood
    ctx.strokeStyle = '#d32f2f';
    ctx.lineWidth = 5;
    ctx.beginPath();
    ctx.arc(x, y - 35, 20, Math.PI, 0);
    ctx.stroke();

    // Rod
    ctx.strokeStyle = '#5d4037';
    ctx.lineWidth = 4;
    ctx.lineCap = 'round';
    ctx.beginPath();
    ctx.moveTo(x + 10, y + 10);
    ctx.lineTo(x + 60, y - 60);
    ctx.stroke();
}

function endGame() {
    gameStarted = false;
    finalScoreVal.innerText = score;
    gameOverModal.classList.remove('hidden');

    // Submit result to backend if user is logged in
    if (window.userID && window.userID !== 'guest') {
        // We don't have a specific endpoint for fishing results yet, 
        // but we could use a generic one if added to the store/handlers.
    }
}

replayBtn.addEventListener('click', startGame);
startGame();
