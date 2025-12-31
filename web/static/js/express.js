document.addEventListener('DOMContentLoaded', () => {
    const gridEl = document.getElementById('gameGrid');
    const piecesContainer = document.getElementById('piecesContainer');
    const currentScoreEl = document.getElementById('currentScore');
    const gameOverModal = document.getElementById('gameOverModal');
    const finalScoreEl = document.getElementById('finalScore');
    const restartBtn = document.getElementById('restartBtn');

    const GRID_SIZE = 10;
    let grid = Array(GRID_SIZE).fill().map(() => Array(GRID_SIZE).fill(null));
    let score = 0;

    // Particle System for Effects
    class Particle {
        constructor(x, y, color) {
            this.x = x;
            this.y = y;
            this.color = color;
            this.size = Math.random() * 6 + 2;
            this.speedX = (Math.random() - 0.5) * 10;
            this.speedY = (Math.random() - 0.5) * 10;
            this.gravity = 0.2;
            this.opacity = 1;
            this.life = 1;
            this.decay = Math.random() * 0.02 + 0.02;
        }

        update() {
            this.x += this.speedX;
            this.y += this.speedY;
            this.speedY += this.gravity;
            this.life -= this.decay;
            this.opacity = Math.max(0, this.life);
        }

        draw(ctx) {
            ctx.fillStyle = this.color;
            ctx.globalAlpha = this.opacity;
            ctx.beginPath();
            ctx.arc(this.x, this.y, this.size, 0, Math.PI * 2);
            ctx.fill();
        }
    }

    class ParticleSystem {
        constructor() {
            this.canvas = document.createElement('canvas');
            this.canvas.style.position = 'fixed';
            this.canvas.style.top = '0';
            this.canvas.style.left = '0';
            this.canvas.style.width = '100%';
            this.canvas.style.height = '100%';
            this.canvas.style.pointerEvents = 'none';
            this.canvas.style.zIndex = '100';
            document.body.appendChild(this.canvas);
            this.ctx = this.canvas.getContext('2d');
            this.particles = [];
            this.resize();
            window.addEventListener('resize', () => this.resize());
            this.animate();
        }

        resize() {
            this.canvas.width = window.innerWidth;
            this.canvas.height = window.innerHeight;
        }

        createBurst(x, y, color, count = 15) {
            for (let i = 0; i < count; i++) {
                this.particles.push(new Particle(x, y, color));
            }
        }

        animate() {
            this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
            for (let i = this.particles.length - 1; i >= 0; i--) {
                const p = this.particles[i];
                p.update();
                if (p.life <= 0) {
                    this.particles.splice(i, 1);
                } else {
                    p.draw(this.ctx);
                }
            }
            requestAnimationFrame(() => this.animate());
        }
    }

    const fx = new ParticleSystem();

    function showComboFeedback(count, x, y) {
        const t = window.translations || {
            doubleClash: "DOUBLE CLASH!",
            tripleClash: "TRIPLE CLASH!",
            quadClash: "QUADRUPLE CLASH!",
            megaClash: "MEGA CLASH!"
        };
        const messages = ["", "", t.doubleClash, t.tripleClash, t.quadClash, t.megaClash];
        const msg = messages[Math.min(count, messages.length - 1)];
        if (!msg) return;

        // Extra VFX for higher combos
        if (count >= 3) {
            for (let i = 0; i < (count - 1); i++) {
                setTimeout(() => {
                    fx.createBurst(x + (Math.random() - 0.5) * 100, y + (Math.random() - 0.5) * 100, 'var(--gold)', 20);
                    fx.createBurst(x + (Math.random() - 0.5) * 100, y + (Math.random() - 0.5) * 100, 'var(--accent)', 15);
                }, i * 200);
            }
        }

        const el = document.createElement('div');
        el.className = 'combo-text';
        el.innerText = msg;
        el.style.left = x + 'px';
        el.style.top = y + 'px';
        document.body.appendChild(el);
        setTimeout(() => el.remove(), 1000);
    }

    // Piece definitions (Classic Block Puzzle shapes)
    const SHAPES = [
        // 1x1
        { id: '1x1', shape: [[1]], color: 'c-gold' },
        // 2x1, 1x2
        { id: '2x1', shape: [[1, 1]], color: 'c-blue' },
        { id: '1x2', shape: [[1], [1]], color: 'c-blue' },
        // 2x2
        { id: '2x2', shape: [[1, 1], [1, 1]], color: 'c-green' },
        // L shapes
        { id: 'L2', shape: [[1, 0], [1, 1]], color: 'c-red' },
        { id: 'L3', shape: [[1, 1, 1], [1, 0, 0]], color: 'c-purple' },
        // I shapes
        { id: 'I3', shape: [[1, 1, 1]], color: 'c-blue' },
        { id: 'I3v', shape: [[1], [1], [1]], color: 'c-blue' },
        { id: 'I4', shape: [[1, 1, 1, 1]], color: 'c-gold' },
        { id: 'I4v', shape: [[1], [1], [1], [1]], color: 'c-gold' },
    ];

    let currentPieces = [];

    // Initialize Grid UI
    function initGrid() {
        gridEl.innerHTML = '';
        for (let r = 0; r < GRID_SIZE; r++) {
            for (let c = 0; c < GRID_SIZE; c++) {
                const cell = document.createElement('div');
                cell.classList.add('grid-cell');
                cell.dataset.row = r;
                cell.dataset.col = c;
                gridEl.appendChild(cell);
            }
        }
    }

    function renderGridState() {
        const cells = gridEl.children;
        for (let r = 0; r < GRID_SIZE; r++) {
            for (let c = 0; c < GRID_SIZE; c++) {
                const index = r * GRID_SIZE + c;
                const cell = cells[index];
                cell.className = 'grid-cell'; // reset
                if (grid[r][c]) {
                    cell.classList.add('filled', grid[r][c]); // add color class
                }
            }
        }
    }

    function spawnPieces() {
        piecesContainer.innerHTML = '';
        currentPieces = [];
        for (let i = 0; i < 3; i++) {
            const template = SHAPES[Math.floor(Math.random() * SHAPES.length)];
            const piece = createPieceElement(template, i);
            piecesContainer.appendChild(piece);
            currentPieces.push({ ...template, element: piece, index: i, used: false });
        }
        checkGameOver();
    }

    function createPieceElement(template, index) {
        const container = document.createElement('div');
        container.classList.add('piece-draggable');
        container.dataset.index = index;

        // Create mini grid for piece
        const rows = template.shape.length;
        const cols = template.shape[0].length;

        container.style.display = 'grid';
        container.style.gridTemplateRows = `repeat(${rows}, 15px)`;
        container.style.gridTemplateColumns = `repeat(${cols}, 15px)`;
        container.style.gap = '2px';

        template.shape.forEach(row => {
            row.forEach(cell => {
                const block = document.createElement('div');
                if (cell) {
                    block.classList.add('block', template.color);
                }
                container.appendChild(block);
            });
        });

        setupDrag(container, template);
        return container;
    }

    let draggedPiece = null;
    let dragClone = null;
    let dragOffset = { x: 0, y: 0 };
    let dragTemplate = null;

    function setupDrag(el, template) {
        // Touch events
        el.addEventListener('touchstart', handleStart, { passive: false });
        // Mouse events
        el.addEventListener('mousedown', handleStart);

        function handleStart(e) {
            if (el.classList.contains('used')) return;
            e.preventDefault();

            const startX = e.touches ? e.touches[0].clientX : e.clientX;
            const startY = e.touches ? e.touches[0].clientY : e.clientY;

            // Calculate offset from center of piece to make it "snap" under finger properly
            // Ideally we pick up from where we touched
            const rect = el.getBoundingClientRect();
            dragOffset.x = startX - rect.left;
            dragOffset.y = startY - rect.top;

            draggedPiece = el;
            dragTemplate = template;

            // Create a clone to drag around
            dragClone = el.cloneNode(true);
            dragClone.style.position = 'fixed';
            dragClone.style.zIndex = '1000';
            dragClone.style.pointerEvents = 'none';
            dragClone.style.width = rect.width + 'px';
            dragClone.style.height = rect.height + 'px';
            // Scale up slightly for effect
            dragClone.style.transform = 'scale(1.2)';
            dragClone.style.opacity = '0.9';

            document.body.appendChild(dragClone);
            moveClone(startX, startY);

            // Highlight listeners
            document.addEventListener('touchmove', handleMove, { passive: false });
            document.addEventListener('touchend', handleEnd);
            document.addEventListener('mousemove', handleMove);
            document.addEventListener('mouseup', handleEnd);
        }
    }

    function handleMove(e) {
        if (!dragClone) return;
        e.preventDefault();
        const x = e.touches ? e.touches[0].clientX : e.clientX;
        const y = e.touches ? e.touches[0].clientY : e.clientY;
        moveClone(x, y);

        // Preview logic (optional, complicated for grid)
        highlightHover(x, y);
    }

    function moveClone(x, y) {
        // Center the touch/mouse on the clone minus offset? 
        // Or actually, let's lift it up a bit so finger doesn't cover it (mobile friendly)
        const liftY = 50;
        dragClone.style.left = (x - dragOffset.x) + 'px';
        dragClone.style.top = (y - dragOffset.y - liftY) + 'px';
    }

    function handleEnd(e) {
        if (!dragClone) return;

        const x = e.changedTouches ? e.changedTouches[0].clientX : e.clientX;
        const y = e.changedTouches ? e.changedTouches[0].clientY : e.clientY;

        // Check drop
        // We need to account for the liftY we added in moveClone to find "true" drops
        const liftY = 50;
        attemptDrop(x, y - liftY);

        // Cleanup
        dragClone.remove();
        dragClone = null;
        draggedPiece = null;
        dragTemplate = null;

        document.removeEventListener('touchmove', handleMove);
        document.removeEventListener('touchend', handleEnd);
        document.removeEventListener('mousemove', handleMove);
        document.removeEventListener('mouseup', handleEnd);

        clearHighlights();
    }

    function highlightHover(x, y) {
        // Simplification: clear previous highlights
        clearHighlights();

        const cell = getCellFromPoint(x, y - 50); // liftY
        if (!cell) return;

        const r = parseInt(cell.dataset.row);
        const c = parseInt(cell.dataset.col);

        if (canPlace(dragTemplate.shape, r, c)) {
            // Visualize placement
            drawPreview(dragTemplate.shape, r, c, 'rgba(255, 255, 255, 0.3)');
        }
    }

    function clearHighlights() {
        const cells = gridEl.getElementsByClassName('grid-cell');
        for (let cell of cells) {
            cell.style.boxShadow = 'none';
            cell.style.borderColor = ''; // reset
        }
    }

    function getCellFromPoint(x, y) {
        // Find grid cell under point
        // We use document.elementFromPoint, but we need to hide the dragClone first? 
        // dragClone has pointer-events: none so it should be fine.
        const el = document.elementFromPoint(x, y);
        if (el && el.classList.contains('grid-cell')) {
            return el;
        }
        return null;
    }

    function canPlace(shape, row, col) {
        for (let i = 0; i < shape.length; i++) {
            for (let j = 0; j < shape[0].length; j++) {
                if (shape[i][j]) {
                    if (row + i >= GRID_SIZE || col + j >= GRID_SIZE || grid[row + i][col + j]) {
                        return false;
                    }
                }
            }
        }
        return true;
    }

    function drawPreview(shape, row, col, color) {
        for (let i = 0; i < shape.length; i++) {
            for (let j = 0; j < shape[0].length; j++) {
                if (shape[i][j]) {
                    const idx = (row + i) * GRID_SIZE + (col + j);
                    const cell = gridEl.children[idx];
                    cell.style.boxShadow = `inset 0 0 10px ${color}`;
                }
            }
        }
    }

    function attemptDrop(x, y) {
        const cell = getCellFromPoint(x, y);
        if (!cell) return;

        const r = parseInt(cell.dataset.row);
        const c = parseInt(cell.dataset.col);

        if (canPlace(dragTemplate.shape, r, c)) {
            placePiece(dragTemplate, r, c);
            draggedPiece.style.opacity = '0';
            draggedPiece.style.pointerEvents = 'none';

            // Mark as used
            const idx = parseInt(draggedPiece.dataset.index);
            currentPieces[idx].used = true;

            // Check if all used
            if (currentPieces.every(p => p.used)) {
                setTimeout(spawnPieces, 500);
            } else {
                checkGameOver();
            }
        }
    }

    function placePiece(template, row, col) {
        for (let i = 0; i < template.shape.length; i++) {
            for (let j = 0; j < template.shape[0].length; j++) {
                if (template.shape[i][j]) {
                    grid[row + i][col + j] = template.color;
                }
            }
        }

        score += 10;
        updateScore();
        renderGridState();

        // Burst effect on placement
        const cell = gridEl.children[row * GRID_SIZE + col];
        const rect = cell.getBoundingClientRect();
        fx.createBurst(rect.left + rect.width / 2, rect.top + rect.height / 2, 'var(--gold)', 5);

        checkLines();
    }

    function checkLines() {
        let linesCleared = 0;
        let rowsToClear = [];
        let colsToClear = [];

        // Check Rows
        for (let r = 0; r < GRID_SIZE; r++) {
            if (grid[r].every(cell => cell !== null)) {
                rowsToClear.push(r);
            }
        }

        // Check Cols
        for (let c = 0; c < GRID_SIZE; c++) {
            let full = true;
            for (let r = 0; r < GRID_SIZE; r++) {
                if (grid[r][c] === null) full = false;
            }
            if (full) colsToClear.push(c);
        }

        // Clear and Animate
        if (rowsToClear.length > 0 || colsToClear.length > 0) {
            const totalLines = rowsToClear.length + colsToClear.length;

            // Screen Shake
            gridEl.classList.add('shake');
            setTimeout(() => gridEl.classList.remove('shake'), 400);

            // Calculate center for combo text
            let avgX = 0, avgY = 0, count = 0;

            rowsToClear.forEach(r => {
                grid[r].forEach((cellValue, c) => {
                    if (cellValue) {
                        const cell = gridEl.children[r * GRID_SIZE + c];
                        const rect = cell.getBoundingClientRect();
                        const colorClass = grid[r][c].split('-')[1];
                        fx.createBurst(rect.left + rect.width / 2, rect.top + rect.height / 2, `var(--${colorClass})`);
                        avgX += rect.left + rect.width / 2;
                        avgY += rect.top + rect.height / 2;
                        count++;
                    }
                });
                grid[r].fill(null);
                score += 100;
            });

            colsToClear.forEach(c => {
                for (let r = 0; r < GRID_SIZE; r++) {
                    if (grid[r][c]) {
                        const cell = gridEl.children[r * GRID_SIZE + c];
                        const rect = cell.getBoundingClientRect();
                        const colorClass = grid[r][c].split('-')[1];
                        fx.createBurst(rect.left + rect.width / 2, rect.top + rect.height / 2, `var(--${colorClass})`);
                        avgX += rect.left + rect.width / 2;
                        avgY += rect.top + rect.height / 2;
                        count++;
                    }
                    grid[r][c] = null;
                }
                score += 100;
            });

            if (totalLines > 1 && count > 0) {
                showComboFeedback(totalLines, avgX / count, avgY / count);
                score += 50 * totalLines;
            }

            setTimeout(renderGridState, 100);
            updateScore();
        }
    }

    function checkGameOver() {
        // If no available pieces, it's not game over yet. Game over only if available pieces cannot fit anywhere.
        const available = currentPieces.filter(p => !p.used);
        if (available.length === 0) return;

        let possibleMove = false;

        for (let p of available) {
            for (let r = 0; r < GRID_SIZE; r++) {
                for (let c = 0; c < GRID_SIZE; c++) {
                    if (canPlace(p.shape, r, c)) {
                        possibleMove = true;
                        break;
                    }
                }
                if (possibleMove) break;
            }
            if (possibleMove) break;
        }

        if (!possibleMove) {
            setTimeout(() => {
                finalScoreEl.innerText = score;
                gameOverModal.classList.remove('hidden');
            }, 500);
        }
    }

    function updateScore() {
        currentScoreEl.innerText = score;
    }

    restartBtn.addEventListener('click', () => {
        grid = Array(GRID_SIZE).fill().map(() => Array(GRID_SIZE).fill(null));
        score = 0;
        updateScore();
        gameOverModal.classList.add('hidden');
        renderGridState();
        spawnPieces();
    });

    // Start
    initGrid();
    spawnPieces();
});
