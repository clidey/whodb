const { invoke } = window.__TAURI__.tauri;

async function initApp() {
    const loadingContainer = document.querySelector('.loading-container');
    const appFrame = document.getElementById('app-frame');
    const statusText = document.querySelector('.status');
    
    try {
        // Wait for backend to be ready
        statusText.textContent = 'Waiting for backend to start...';
        await invoke('wait_for_backend');
        
        // Get backend URL
        const backendUrl = await invoke('get_backend_url');
        
        // Load the app
        statusText.textContent = 'Loading application...';
        appFrame.src = backendUrl;
        
        // Wait for iframe to load
        appFrame.onload = () => {
            loadingContainer.style.display = 'none';
            appFrame.style.display = 'block';
        };
        
        // Handle iframe errors
        appFrame.onerror = (error) => {
            showError('Failed to load application: ' + error);
        };
        
    } catch (error) {
        showError(error);
    }
}

function showError(error) {
    const loadingContainer = document.querySelector('.loading-container');
    const spinner = loadingContainer.querySelector('.spinner');
    const status = loadingContainer.querySelector('.status');
    
    spinner.style.display = 'none';
    status.innerHTML = `
        <div class="error">
            <strong>Error:</strong> ${error}
            <br><br>
            Please check that the backend is properly configured and try restarting the application.
        </div>
    `;
}

// Initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initApp);
} else {
    initApp();
}