function handleDownload(btn) {
    const original = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = '...';
    setTimeout(() => {
        if (btn.disabled) {
            btn.disabled = false;
            btn.innerHTML = original;
        }
    }, 10000);
}
