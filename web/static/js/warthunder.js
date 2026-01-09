// warthunder.js - Political Domination Simulator Client
const API_BASE = '/api/warthunder';
let gameState = null;
let selectedCountry = null;
let currentTab = 'events';
let updateInterval = null;

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    console.log('Game initializing...');
    loadGameState();
    setupMapInteractions();
});

// Load game state from server
async function loadGameState() {
    try {
        const response = await fetch(`${API_BASE}?userID=${window.USER_ID}`);
        const data = await response.json();

        if (data.status === 'selection') {
            showView('selection');
            populateCountrySelection(data.countries);
        } else if (data.status === 'playing') {
            gameState = data.game;
            showView('dashboard');
            updateDashboard();
            startAutoUpdate();
        }
    } catch (error) {
        console.error('Failed to load game:', error);
        showNotification('⚠️ Connection error', 'error');
    }
}

// Setup interactive map
function setupMapInteractions() {
    const countries = document.querySelectorAll('.country');
    countries.forEach(country => {
        country.addEventListener('click', () => {
            selectCountryForStart(country.id);
        });

        country.addEventListener('mouseenter', (e) => {
            country.style.filter = 'brightness(1.5)';
        });

        country.addEventListener('mouseleave', (e) => {
            country.style.filter = 'brightness(1)';
        });
    });
}

// Populate country selection data
function populateCountrySelection(countries) {
    countries.forEach(country => {
        const pathElement = document.getElementById(country.id);
        if (pathElement) {
            pathElement.setAttribute('data-name', country.name);
            pathElement.setAttribute('data-pop', country.population);
            pathElement.setAttribute('data-eco', country.economy);
            pathElement.setAttribute('data-mil', country.military);
            pathElement.setAttribute('data-stab', country.stability);
        }
    });
}

// Select country from map
function selectCountryForStart(countryId) {
    const pathElement = document.getElementById(countryId);
    if (!pathElement) return;

    selectedCountry = countryId;

    // Highlight selected
    document.querySelectorAll('.country').forEach(c => {
        c.style.strokeWidth = '2';
    });
    pathElement.style.strokeWidth = '4';

    // Show selection panel
    const panel = document.getElementById('selection-panel');
    panel.classList.remove('hidden');

    document.getElementById('sel-name').textContent = pathElement.getAttribute('data-name');
    document.getElementById('sel-pop').textContent = formatNumber(pathElement.getAttribute('data-pop'));
    document.getElementById('sel-eco').textContent = `$${pathElement.getAttribute('data-eco')}B`;
    document.getElementById('sel-mil').textContent = pathElement.getAttribute('data-mil');
    document.getElementById('sel-stab').textContent = `${pathElement.getAttribute('data-stab')}%`;
}

// Start game with selected country
document.getElementById('btn-start').addEventListener('click', async () => {
    if (!selectedCountry) return;

    try {
        const response = await fetch(`${API_BASE}?userID=${window.USER_ID}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                action: 'start',
                payload: selectedCountry
            })
        });

        const data = await response.json();
        if (data.status === 'started') {
            gameState = data.game;
            showView('dashboard');
            updateDashboard();
            startAutoUpdate();
            showNotification('🎮 Game Started! Lead your nation to glory!', 'success');
        }
    } catch (error) {
        console.error('Failed to start game:', error);
        showNotification('⚠️ Failed to start game', 'error');
    }
});

// Switch between views
function showView(viewName) {
    document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
    document.getElementById(`view-${viewName}`).classList.add('active');
}

// Update entire dashboard
function updateDashboard() {
    if (!gameState) return;

    const player = gameState.countries[gameState.playerCountry];
    if (!player) return;

    // Update header
    document.getElementById('dashboard-country').textContent = player.name;
    document.getElementById('turn-number').textContent = gameState.turn;

    // Update resources
    document.getElementById('res-economy').textContent = `$${player.economy.toFixed(1)}B`;
    document.getElementById('res-military').textContent = Math.round(player.military);
    document.getElementById('res-stability').textContent = `${Math.round(player.stability)}%`;
    document.getElementById('res-approval').textContent = `${Math.round(player.approvalRating)}%`;
    document.getElementById('res-tech').textContent = Math.round(player.techLevel);
    document.getElementById('res-corruption').textContent = `${Math.round(player.corruption)}%`;

    // Update resource stockpiles
    document.getElementById('res-oil').textContent = Math.round(player.resources.oil);
    document.getElementById('res-food').textContent = Math.round(player.resources.food);
    document.getElementById('res-tech-resource').textContent = Math.round(player.resources.tech);

    // Update global tension
    const tension = Math.round(gameState.globalTension);
    document.getElementById('tension-value').textContent = `${tension}%`;
    document.getElementById('tension-fill').style.width = `${tension}%`;

    // Update event log
    updateEventLog();

    // Update alliances
    updateAlliances();

    // Update sanctions
    updateSanctions();

    // Update victory progress
    updateVictoryProgress();

    // Update active tab
    updateActiveTab();

    // Check game over
    if (gameState.gameOver) {
        showVictoryScreen();
    }
}

// Update event log
function updateEventLog() {
    const log = document.getElementById('event-log');
    log.innerHTML = '';

    gameState.events.slice(0, 20).forEach(event => {
        const li = document.createElement('li');
        li.textContent = event;
        log.appendChild(li);
    });
}

// Update alliances display
function updateAlliances() {
    const player = gameState.countries[gameState.playerCountry];
    const content = document.getElementById('alliances-content');

    if (!player.alliances || player.alliances.length === 0) {
        content.innerHTML = '<p style="opacity: 0.6; font-size: 0.9em;">No alliances yet</p>';
    } else {
        content.innerHTML = '';
        player.alliances.forEach(allyId => {
            const ally = gameState.countries[allyId];
            if (ally) {
                const tag = document.createElement('div');
                tag.className = 'alliance-tag';
                tag.textContent = ally.name;
                content.appendChild(tag);
            }
        });
    }
}

// Update sanctions display
function updateSanctions() {
    const player = gameState.countries[gameState.playerCountry];
    const content = document.getElementById('sanctions-content');

    if (!player.sanctions || player.sanctions.length === 0) {
        content.innerHTML = '<p style="opacity: 0.6; font-size: 0.9em;">No sanctions</p>';
    } else {
        content.innerHTML = '';
        player.sanctions.forEach(sanctionerId => {
            const sanctioner = gameState.countries[sanctionerId];
            if (sanctioner) {
                const tag = document.createElement('div');
                tag.className = 'sanction-tag';
                tag.textContent = sanctioner.name;
                content.appendChild(tag);
            }
        });
    }

    // UN Sanctions
    if (gameState.unSanctions && gameState.unSanctions[gameState.playerCountry]) {
        const unTag = document.createElement('div');
        unTag.className = 'sanction-tag';
        unTag.style.borderColor = '#FF6B6B';
        unTag.textContent = `🏛️ UN (${gameState.unSanctions[gameState.playerCountry]} turns left)`;
        content.appendChild(unTag);
    }
}

// Update victory progress bars
function updateVictoryProgress() {
    const player = gameState.countries[gameState.playerCountry];

    // Count alive countries
    let aliveCount = 0;
    Object.values(gameState.countries).forEach(c => {
        if (!c.isEliminated) aliveCount++;
    });

    // Domination (only you remain)
    const dominationPercent = ((11 - aliveCount) / 10) * 100;
    document.getElementById('victory-domination').textContent = `${Math.round(dominationPercent)}%`;
    document.getElementById('victory-domination-bar').style.width = `${dominationPercent}%`;

    // Economic (reach $50,000B)
    const economicPercent = Math.min((player.economy / 50000) * 100, 100);
    document.getElementById('victory-economic').textContent = `${Math.round(economicPercent)}%`;
    document.getElementById('victory-economic-bar').style.width = `${economicPercent}%`;

    // Diplomatic (6+ alliances)
    const allianceCount = player.alliances ? player.alliances.length : 0;
    const diplomaticPercent = (allianceCount / 6) * 100;
    document.getElementById('victory-diplomatic').textContent = `${allianceCount}/6`;
    document.getElementById('victory-diplomatic-bar').style.width = `${diplomaticPercent}%`;

    // Tech (level 100 + 1000 tech resources)
    const techPercent = Math.min(((player.techLevel / 100) * 50 + (player.resources.tech / 1000) * 50), 100);
    document.getElementById('victory-tech').textContent = `${Math.round(techPercent)}%`;
    document.getElementById('victory-tech-bar').style.width = `${techPercent}%`;
}

// Show victory screen
function showVictoryScreen() {
    const overlay = document.getElementById('victory-overlay');
    const title = document.getElementById('victory-title');
    const message = document.getElementById('victory-message');

    const victoryTypes = {
        'domination': '🌍 You conquered the world!',
        'economic': '💰 Your economy dominates!',
        'diplomatic': '🤝 You united the world!',
        'technological': '🔬 You lead humanity forward!',
        'defeat': '💀 You have been overthrown...'
    };

    title.textContent = gameState.victoryType === 'defeat' ? '💀 DEFEAT' : '🏆 VICTORY!';
    message.textContent = victoryTypes[gameState.victoryType] || 'Game Over';

    overlay.classList.add('active');
}

// Tab switching
function switchTab(tabName) {
    currentTab = tabName;

    // Update tab buttons
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    event.target.classList.add('active');

    // Update tab content
    document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
    document.getElementById(`tab-${tabName}`).classList.add('active');

    updateActiveTab();
}

// Update content of active tab
function updateActiveTab() {
    switch (currentTab) {
        case 'events':
            // Already updated in updateEventLog
            break;
        case 'war':
            updateWarRoom();
            break;
        case 'diplomacy':
            updateDiplomacy();
            break;
        case 'espionage':
            updateEspionage();
            break;
        case 'world':
            updateWorldMap();
            break;
    }
}

// Update War Room tab
function updateWarRoom() {
    const container = document.getElementById('war-targets');
    container.innerHTML = '';

    Object.values(gameState.countries).forEach(country => {
        if (country.id === gameState.playerCountry || country.isEliminated) return;

        const card = createCountryCard(country, 'war');
        container.appendChild(card);
    });
}

// Update Diplomacy tab
function updateDiplomacy() {
    const container = document.getElementById('diplo-targets');
    container.innerHTML = '';

    Object.values(gameState.countries).forEach(country => {
        if (country.id === gameState.playerCountry || country.isEliminated) return;

        const card = createCountryCard(country, 'diplomacy');
        container.appendChild(card);
    });
}

// Update Espionage tab
function updateEspionage() {
    const container = document.getElementById('espionage-targets');
    container.innerHTML = '';

    Object.values(gameState.countries).forEach(country => {
        if (country.id === gameState.playerCountry || country.isEliminated) return;

        const card = createCountryCard(country, 'espionage');
        container.appendChild(card);
    });
}

// Update World Map tab
function updateWorldMap() {
    const container = document.getElementById('world-overview');
    container.innerHTML = '';

    Object.values(gameState.countries).forEach(country => {
        const card = createCountryCard(country, 'overview');
        container.appendChild(card);
    });
}

// Create country card for different contexts
function createCountryCard(country, context) {
    const player = gameState.countries[gameState.playerCountry];
    const relation = country.relations ? country.relations[gameState.playerCountry] || 0 : 0;

    const card = document.createElement('div');
    card.className = 'country-card';
    if (country.isEliminated) card.classList.add('eliminated');
    if (country.isPlayer) card.style.borderColor = '#FFD700';

    const relationPercent = ((relation + 100) / 200) * 100;
    const relationClass = relation >= 0 ? 'positive' : 'negative';

    card.innerHTML = `
        <div class="country-card-header">
            <div class="country-flag" style="background: ${country.color}"></div>
            <div>
                <h3>${country.name}</h3>
                <div style="font-size: 0.85em; opacity: 0.7;">${country.government} - ${country.ideology}</div>
            </div>
        </div>

        ${country.isEliminated ? '<div style="text-align: center; color: #f5576c; font-weight: bold; margin: 10px 0;">❌ ELIMINATED</div>' : ''}

        <div class="country-stats">
            <div><span>💰 Economy:</span> <span>$${country.economy.toFixed(1)}B</span></div>
            <div><span>⚔️ Military:</span> <span>${Math.round(country.military)}</span></div>
            <div><span>📊 Stability:</span> <span>${Math.round(country.stability)}%</span></div>
            <div><span>📈 Approval:</span> <span>${Math.round(country.approvalRating)}%</span></div>
            <div><span>🔬 Tech:</span> <span>${Math.round(country.techLevel)}</span></div>
            ${country.alliances && country.alliances.length > 0 ? `<div><span>🛡️ Allies:</span> <span>${country.alliances.length}</span></div>` : ''}
        </div>

        ${!country.isEliminated && !country.isPlayer ? `
            <div>
                <div style="font-size: 0.85em; margin-top: 10px; margin-bottom: 5px;">Relations: ${relation > 0 ? '+' : ''}${Math.round(relation)}</div>
                <div class="relation-bar">
                    <div class="relation-fill ${relationClass}" style="width: ${relationPercent}%"></div>
                </div>
            </div>
        ` : ''}

        <div class="country-actions" id="actions-${country.id}"></div>
    `;

    // Add context-specific actions
    const actionsContainer = card.querySelector(`#actions-${country.id}`);

    if (!country.isEliminated && !country.isPlayer) {
        if (context === 'war') {
            const attackBtn = document.createElement('button');
            attackBtn.textContent = '⚔️ Attack';
            attackBtn.style.background = 'rgba(245, 87, 108, 0.3)';
            attackBtn.style.borderColor = '#f5576c';
            attackBtn.onclick = () => performAction('attack', country.id);
            actionsContainer.appendChild(attackBtn);
        }

        if (context === 'diplomacy') {
            const diplomacyBtn = document.createElement('button');
            diplomacyBtn.textContent = '🤝 Improve Relations';
            diplomacyBtn.onclick = () => performAction('diplomat', country.id);
            actionsContainer.appendChild(diplomacyBtn);

            if (relation >= 50 && !player.alliances.includes(country.id)) {
                const allianceBtn = document.createElement('button');
                allianceBtn.textContent = '🛡️ Form Alliance';
                allianceBtn.style.background = 'rgba(76, 175, 80, 0.3)';
                allianceBtn.style.borderColor = '#4CAF50';
                allianceBtn.onclick = () => performAction('formAlliance', country.id);
                actionsContainer.appendChild(allianceBtn);
            }

            const sanctionBtn = document.createElement('button');
            sanctionBtn.textContent = '📛 Sanction';
            sanctionBtn.style.background = 'rgba(255, 152, 0, 0.3)';
            sanctionBtn.onclick = () => performAction('imposeSanctions', country.id);
            actionsContainer.appendChild(sanctionBtn);
        }

        if (context === 'espionage') {
            const spyBtn = document.createElement('button');
            spyBtn.textContent = '🕵️ Espionage ($50B)';
            spyBtn.onclick = () => performAction('espionage', country.id);
            actionsContainer.appendChild(spyBtn);
        }
    }

    return card;
}

// Perform game action
async function performAction(action, targetId = null) {
    try {
        const payload = {
            action: action,
            payload: targetId || ''
        };

        const response = await fetch(`${API_BASE}?userID=${window.USER_ID}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        const data = await response.json();

        if (data.message) {
            const messageType = data.message.includes('success') || data.message.includes('Victory') ? 'success' :
                data.message.includes('Insufficient') || data.message.includes('Defeat') ? 'error' : 'info';
            showNotification(data.message, messageType);
        }

        if (data.game) {
            gameState = data.game;
            updateDashboard();
        }
    } catch (error) {
        console.error('Action failed:', error);
        showNotification('⚠️ Action failed', 'error');
    }
}

// Quick actions from sidebar
async function gameAction(action) {
    await performAction(action);
}

// Auto-update game state
function startAutoUpdate() {
    if (updateInterval) clearInterval(updateInterval);

    updateInterval = setInterval(async () => {
        try {
            const response = await fetch(`${API_BASE}?userID=${window.USER_ID}`);
            const data = await response.json();

            if (data.status === 'playing' && data.game) {
                gameState = data.game;
                updateDashboard();
            }
        } catch (error) {
            console.error('Auto-update failed:', error);
        }
    }, 5000); // Update every 5 seconds
}

// Show notification
function showNotification(message, type = 'info') {
    const notification = document.createElement('div');
    notification.className = 'notification';
    notification.textContent = message;

    if (type === 'error') notification.style.borderLeftColor = '#f5576c';
    if (type === 'success') notification.style.borderLeftColor = '#4CAF50';

    document.getElementById('notification-area').appendChild(notification);

    setTimeout(() => {
        notification.style.animation = 'slideOutRight 0.3s ease';
        setTimeout(() => notification.remove(), 300);
    }, 4000);
}

// Utility: Format large numbers
function formatNumber(num) {
    num = parseInt(num);
    if (num >= 1000000000) return (num / 1000000000).toFixed(1) + 'B';
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
}

// Add slideOutRight animation to CSS
const style = document.createElement('style');
style.textContent = `
    @keyframes slideOutRight {
        from { opacity: 1; transform: translateX(0); }
        to { opacity: 0; transform: translateX(100px); }
    }
`;
document.head.appendChild(style);