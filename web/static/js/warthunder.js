(function() {
    const API_URL = '/api/warthunder';
    
    let gameState = null;
    let selectedCountryID = null;
    let countryData = []; // Loaded from server

    // DOM Elements
    const views = {
        selection: document.getElementById('view-selection'),
        dashboard: document.getElementById('view-dashboard')
    };

    const ui = {
        map: document.getElementById('world-map'),
        selectionPanel: document.getElementById('selection-panel'),
        tooltip: document.getElementById('selection-tooltip'),
        
        // Selection
        selName: document.getElementById('sel-name'),
        selPop: document.getElementById('sel-pop'),
        selEco: document.getElementById('sel-eco'),
        selMil: document.getElementById('sel-mil'),
        btnStart: document.getElementById('btn-start'),

        // Dashboard
        countryName: document.getElementById('dashboard-country'),
        resEconomy: document.getElementById('res-economy'),
        resMilitary: document.getElementById('res-military'),
        resStability: document.getElementById('res-stability'),
        eventLog: document.getElementById('event-log'),
        warTargets: document.getElementById('war-targets'),
        diploTargets: document.getElementById('diplo-targets'),
        
        navBtns: document.querySelectorAll('.nav-btn'),
        tabs: document.querySelectorAll('.tab-content')
    };

    // --- Init ---
    async function init() {
        const res = await fetch(`${API_URL}?userID=${window.USER_ID}`);
        if (!res.ok) return; // Handle error
        const data = await res.json();

        if (data.status === 'selection') {
            countryData = data.countries;
            setupMap();
            switchView('selection');
        } else if (data.status === 'playing') {
            gameState = data.game;
            renderGame();
            switchView('dashboard');
        }
    }

    // --- Map Logic ---
    function setupMap() {
        // Color known countries
        countryData.forEach(c => {
            const el = document.getElementById(c.id);
            if (el) {
                el.dataset.color = c.color;
            }
        });

        ui.map.addEventListener('mouseover', (e) => {
            if (e.target.classList.contains('country')) {
                const id = e.target.id;
                const data = countryData.find(c => c.id === id);
                if (data) {
                    ui.tooltip.style.display = 'block';
                    ui.tooltip.textContent = data.name;
                    e.target.style.fill = data.color;
                }
            }
        });

        ui.map.addEventListener('mousemove', (e) => {
            ui.tooltip.style.left = (e.clientX + 10) + 'px';
            ui.tooltip.style.top = (e.clientY + 10) + 'px';
        });

        ui.map.addEventListener('mouseout', (e) => {
            if (e.target.classList.contains('country')) {
                ui.tooltip.style.display = 'none';
                if (e.target.id !== selectedCountryID) {
                    e.target.style.fill = ''; // Reset
                }
            }
        });

        ui.map.addEventListener('click', (e) => {
            if (e.target.classList.contains('country')) {
                selectCountry(e.target.id);
            }
        });
    }

    function selectCountry(id) {
        selectedCountryID = id;
        
        // Highlight logic
        document.querySelectorAll('.country').forEach(el => el.classList.remove('selected'));
        document.getElementById(id).classList.add('selected');

        const data = countryData.find(c => c.id === id);
        if (data) {
            ui.selName.textContent = data.name;
            ui.selPop.textContent = formatNumber(data.population);
            ui.selEco.textContent = `$${data.economy}B`;
            ui.selMil.textContent = data.military;
            
            ui.selectionPanel.classList.remove('hidden');
        }
    }

    ui.btnStart.onclick = async () => {
        if (!selectedCountryID) return;
        
        const res = await fetch(API_URL, {
            method: 'POST',
            body: JSON.stringify({ action: 'start', payload: selectedCountryID })
        });
        const data = await res.json();
        if (data.status === 'started') {
            gameState = data.game;
            renderGame();
            switchView('dashboard');
        }
    };

    // --- Gameplay Logic ---
    function renderGame() {
        if (!gameState) return;

        const player = gameState.countries[gameState.playerCountry];
        
        ui.countryName.textContent = player.name;
        ui.resEconomy.textContent = `$${player.economy.toFixed(1)}B`;
        ui.resMilitary.textContent = Math.round(player.military);
        ui.resStability.textContent = `${Math.round(player.stability)}%`;

        renderEvents(gameState.events);
        renderTargets();
    }

    function renderEvents(events) {
        ui.eventLog.innerHTML = events.map(e => `<li>${e}</li>`).join('');
    }

    function renderTargets() {
        ui.warTargets.innerHTML = '';
        ui.diploTargets.innerHTML = '';

        Object.values(gameState.countries).forEach(c => {
            if (c.id === gameState.playerCountry || c.isEliminated) return;

            // Attack Card
            const warCard = document.createElement('div');
            warCard.className = 'action-card';
            warCard.innerHTML = `
                <h4>${c.name}</h4>
                <p>Mil: ${Math.round(c.military)}</p>
                <div class="stat">Rel: ${Math.round(c.relations[gameState.playerCountry] || 0)}</div>
                <button class="action-btn attack">ATTACK</button>
            `;
            warCard.querySelector('button').onclick = () => performAction('attack', c.id);
            ui.warTargets.appendChild(warCard);

            // Diplo Card
            const diploCard = document.createElement('div');
            diploCard.className = 'action-card';
            diploCard.innerHTML = `
                 <h4>${c.name}</h4>
                 <p>Eco: $${c.economy}B</p>
                 <div class="stat">Rel: ${Math.round(c.relations[gameState.playerCountry] || 0)}</div>
                 <button class="action-btn diplo">IMPROVE RELATIONS (-$10B)</button>
            `;
            diploCard.querySelector('button').onclick = () => performAction('diplomat', c.id);
            ui.diploTargets.appendChild(diploCard);
        });
    }

    async function performAction(action, targetID) {
        const res = await fetch(API_URL, {
            method: 'POST',
            body: JSON.stringify({ action: action, payload: targetID })
        });
        const data = await res.json();
        
        if (data.status === 'ok' || data.type === 'defeat' || data.type === 'victory') {
            gameState = data.game;
            renderGame();
            
            if (data.message === 'victory') {
                alert('War Won! Territory annexed.');
            } else if (data.message === 'defeat') {
                alert('War Lost! Economy and military suffered.');
            }
        } else {
            alert(data.message || 'Action failed');
        }
    }

    // --- Tabs ---
    ui.navBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            ui.navBtns.forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            
            ui.tabs.forEach(t => t.classList.remove('active'));
            document.getElementById(`tab-${btn.dataset.tab}`).classList.add('active');
        });
    });

    // --- Helpers ---
    function switchView(viewName) {
        Object.values(views).forEach(el => el.classList.remove('active'));
        views[viewName].classList.add('active');
    }

    function formatNumber(num) {
        if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
        return num.toLocaleString();
    }

    init();
})();
