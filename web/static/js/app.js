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

class TagInput {
  constructor(config) {
    this.input = document.getElementById(config.inputId);
    this.hidden = config.hiddenId ? document.getElementById(config.hiddenId) : null;
    this.tagsContainer = document.getElementById(config.tagsId);
    this.suggestionsContainer = document.getElementById(config.suggestionsId);
    this.options = config.options || [];
    this.tags = config.initialValues ? [...config.initialValues] : [];

    if (!this.input || !this.tagsContainer || !this.suggestionsContainer) {
      console.error('TagInput: Required elements not found');
      return;
    }

    this.input._tagInput = this;
    this.init();
  }

  init() {
    this.input.addEventListener('keydown', (e) => this.onKeydown(e));
    this.input.addEventListener('input', (e) => this.onInput(e));
    this.input.addEventListener('focus', (e) => this.onFocus(e));
    this.input.addEventListener('blur', () => setTimeout(() => this.hideSuggestions(), 150));

    document.addEventListener('click', (e) => this.onDocumentClick(e));

    this.renderTags();
  }

  onKeydown(e) {
    if (e.key === 'Enter') {
      e.preventDefault();
      this.addTag(this.input.value);
    } else if (e.key === 'Backspace' && !this.input.value && this.tags.length > 0) {
      this.tags.pop();
      this.renderTags();
    }
  }

  onInput(e) {
    this.showSuggestions(this.input.value, false);
  }

  onFocus(e) {
    this.showSuggestions('', true);
  }

  onDocumentClick(e) {
    const wrapper = this.input.closest('.tag-input-wrapper');
    const inSuggestions = this.suggestionsContainer.contains(e.target);
    if (wrapper && !wrapper.contains(e.target) && !inSuggestions) {
      this.hideSuggestions();
    }
  }

  addTag(value) {
    value = value.trim();
    if (!value) return;

    // Check for duplicates (case-insensitive)
    const exists = this.tags.some(t => t.toLowerCase() === value.toLowerCase());
    if (!exists) {
      this.tags.push(value);
      this.renderTags();
    }
    
    this.input.value = '';
    
    // If input is still focused (e.g., clicking a suggestion kept focus), update the list
    if (document.activeElement === this.input) {
      this.showSuggestions('', true);
    } else {
      this.hideSuggestions();
    }
  }

  removeTag(value) {
    this.tags = this.tags.filter(t => t !== value);
    this.renderTags();
  }

  reset() {
    this.tags = [];
    this.input.value = '';
    this.renderTags();
    this.hideSuggestions();
  }

  renderTags() {
    this.tagsContainer.innerHTML = '';
    this.tags.forEach(tag => {
      const el = document.createElement('span');
      el.className = 'tag';
      el.textContent = tag;

      const rm = document.createElement('span');
      rm.className = 'tag-remove';
      rm.textContent = '×';
      rm.addEventListener('click', () => this.removeTag(tag));

      el.appendChild(rm);
      this.tagsContainer.appendChild(el);
    });

    if (this.hidden) {
      this.hidden.value = this.tags.join(';');
    }
  }

  showSuggestions(query, showAllOnEmpty) {
    const matches = this.options.filter(o => 
      o.toLowerCase().includes(query.toLowerCase()) && 
      !this.tags.some(t => t.toLowerCase() === o.toLowerCase())
    );

    if (matches.length === 0) {
      this.hideSuggestions();
      return;
    }

    if (!query && !showAllOnEmpty) {
      this.hideSuggestions();
      return;
    }

    this.suggestionsContainer.innerHTML = '';
    matches.slice(0, 15).forEach(m => {
      const div = document.createElement('div');
      div.className = 'tag-suggestion';
      div.textContent = m;
      // Use mousedown instead of click to fire before the input's blur event
      div.addEventListener('mousedown', (e) => {
        this.addTag(m);
        this.input.blur(); // Explicitly remove focus, hiding dropdown
      });
      this.suggestionsContainer.appendChild(div);
    });

    this.suggestionsContainer.classList.add('show');
  }

  hideSuggestions() {
    this.suggestionsContainer.classList.remove('show');
  }
}
