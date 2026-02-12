// Initialize Lucide icons with custom attributes
function initIcons(root = document) {
    if (window.lucide) {
        lucide.createIcons({
            root: root,
            attrs: {
                'stroke-width': 2.5,
                'stroke': 'currentColor'
            }
        });
    }
}

// Initialize icons after HTMX content loads
if (typeof htmx !== 'undefined') {
    htmx.onLoad(function (content) {
        // Small delay to ensure DOM is settled
        setTimeout(() => initIcons(content), 0);
    });
}

// Initialize icons on page load
document.addEventListener('DOMContentLoaded', () => initIcons());

/**
 * Handle download button click - shows loading state
 * @param {HTMLButtonElement} btn - The clicked button
 */
function handleDownload(btn) {
    const originalContent = btn.innerHTML;
    btn.disabled = true;

    if (btn.classList.contains('btn-icon')) {
        btn.innerHTML = '<i data-lucide="loader-2" class="animate-spin"></i>';
    } else {
        btn.innerHTML = '<i data-lucide="loader-2" class="animate-spin"></i> Starting...';
    }
    initIcons(btn);

    // Reset button after 10 seconds if still loading
    setTimeout(() => {
        if (btn.disabled && btn.innerHTML.includes('loader-2')) {
            btn.disabled = false;
            btn.innerHTML = originalContent;
            initIcons(btn);
        }
    }, 10000);
}
