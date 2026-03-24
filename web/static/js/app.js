function handleDownload(btn) {
  const originalText = btn.innerText;
  btn.disabled = true;
  btn.setAttribute('data-original', originalText);
  btn.innerText = '...';
}

function queueDownload(e, type, id, btn) {
  e.preventDefault();
  e.stopPropagation();
  handleDownload(btn);
  fetch(`/htmx/download/${type}/${id}`, {
    method: 'POST',
    headers: { 'HX-Request': 'true' }
  }).then(() => {
    // Keep button disabled - don't re-enable
  }).catch(() => {
    btn.disabled = false;
    btn.innerText = btn.getAttribute('data-original') || 'Download';
  });
}
