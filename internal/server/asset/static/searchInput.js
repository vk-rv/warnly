window.searchInput = function(initialTokens, filterCategories, options) {
  return {
    tokens: initialTokens || [],
    inputValue: '',
    showTagSuggestions: false,
    showOperatorDropdown: false,
    showTagMatch: false,
    matchedTag: null,
    activeTokenIndex: -1,
    operatorDropdownPosition: { top: 0, left: 0 },
    tagValues: [],
    customValue: '',
    options: options || {},

    filterCategories: filterCategories || [],

    init() {
      this.originalCategories = [...this.filterCategories];
      this.$watch('tokens', () => {
        window.dispatchEvent(new CustomEvent('tokens-changed', { detail: { tokens: this.tokens } }));
      });


      window.addEventListener('reset-search', () => {
        this.clearAll();
      });
    },

    focusInput() {
      this.$refs.searchInput.focus();
    },

    handleInputClick() {
    if (this.inputValue === '') {
    this.filterCategories = [...this.originalCategories]; 
    this.showTagSuggestions = true;
      this.showTagMatch = false;
      }
    },

    handleInputFocus() {
    if (this.inputValue === '') {
    this.filterCategories = [...this.originalCategories]; 
      this.showTagSuggestions = true;
        this.showTagMatch = false;
      }
    },

    handleInput() {
      if (this.inputValue === '') {
        this.filterCategories = [...this.originalCategories]; 
        this.showTagSuggestions = true;
        this.showTagMatch = false;
        this.matchedTag = null;
      } else {
       
        if (this.isInTagValuesMode()) {
          this.showTagSuggestions = true; 
          this.showTagMatch = false;
        } else {
          this.showTagSuggestions = false;
          this.checkForTagMatch();
        }
      }
    },

    checkForTagMatch() {
      const input = this.inputValue.trim().toLowerCase();
      this.matchedTag = null;
      this.showTagMatch = false;

 
      for (const category of this.filterCategories) {
        for (const item of category.items) {
          if (item.key.toLowerCase() === input) {
            this.matchedTag = item;
            this.showTagMatch = true;
            break;
          }
        }
        if (this.matchedTag) break;
      }
    },

    handleEnterKey() {
      if (this.showTagMatch && this.matchedTag) {
        this.selectMatchedTag();
      } else if (this.inputValue.trim() !== '') {
        if (this.isInTagValuesMode()) {
         
          this.parseAndAddTokenInValuesMode(this.inputValue.trim());
        } else {
          this.addRawTextToken(this.inputValue.trim());
        }
        this.inputValue = '';
      }
      this.closeAllDropdowns();
    },

    handleBackspace() {
      if (this.inputValue === '' && this.tokens.length > 0) {
        this.removeToken(this.tokens.length - 1);
      }
    },

    addToken(key, operator, value) {
      this.tokens.push({
        key,
        operator: operator || 'is',
        value,
        isRawText: false
      });
      this.inputValue = '';
      this.closeAllDropdowns();
    },

    addRawTextToken(value) {
      this.tokens.push({
        key: '',
        operator: '',
        value,
        isRawText: true
      });
    },

    removeToken(index) {
      this.tokens.splice(index, 1);
    },

    openOperatorDropdown(index, event) {
      const tokenElement = event ? event.target : document.querySelector('.tag-pill-operator');
      const rect = tokenElement.getBoundingClientRect();

      this.operatorDropdownPosition = {
        top: rect.bottom + window.scrollY + 2,
        left: rect.left + window.scrollX
      };

      this.activeTokenIndex = index;
      this.showOperatorDropdown = true;
    },

    changeOperator(index, operator) {
      if (this.tokens[index] && !this.tokens[index].isRawText) {
        this.tokens[index].operator = operator;
      }
      this.showOperatorDropdown = false;
    },

    addFilterFromCategory(item) {
     
      if (item.key === item.value) {
        this.loadTagValues(item.key);
      } else {
        this.addToken(item.key, 'is', item.value);
      }
    },

    selectMatchedTag() {
      if (this.matchedTag) {
        this.addFilterFromCategory(this.matchedTag);
        this.inputValue = '';
        this.showTagMatch = false;
        this.matchedTag = null;
      }
    },

    loadTagValues(tag) {
      const url = `/api/search/tag-values?tag=${encodeURIComponent(tag)}&project_name=${encodeURIComponent(this.getProjectName())}&period=${encodeURIComponent(this.getPeriod())}`;
      fetch(url)
        .then(response => response.json())
        .then(data => {
          if (!data || !Array.isArray(data)) {
            console.error('Invalid data received for tag values:', data);
            return;
          }
          this.tagValues = data;
  
          this.filterCategories = [{
            name: `Values for ${tag}`,
            active: true,
            items: data.filter(v => v && v.value).map(v => ({ key: tag, value: v.value }))
          }];
          this.showTagSuggestions = true;
        })
        .catch(error => {
          console.error('Failed to load tag values:', error);
          this.filterCategories = [...this.originalCategories];
          this.showTagSuggestions = true;
        });
    },

    getProjectName() {
      const urlParams = new URLSearchParams(window.location.search);
      return urlParams.get('project_name') || '';
    },

    getPeriod() {
      if (this.options.period) {
        return this.options.period;
      }
      const urlParams = new URLSearchParams(window.location.search);
      return urlParams.get('period') || '14d';
    },

    setActiveCategory(index) {
      this.filterCategories.forEach((cat, i) => {
        cat.active = i === index;
      });
    },

    closeAllDropdowns() {
      this.showTagSuggestions = false;
      this.showOperatorDropdown = false;
      this.showTagMatch = false;
      this.matchedTag = null;
      this.customValue = '';
    },

    isInTagValuesMode() {
      return this.filterCategories.length > 0 &&
             this.filterCategories[0].name &&
             this.filterCategories[0].name.startsWith('Values for ');
    },

    getCurrentTag() {
      if (this.isInTagValuesMode()) {
        const name = this.filterCategories[0].name;
        return name.replace('Values for ', '');
      }
      return null;
    },

    addCustomValue() {
      if (this.customValue.trim() !== '') {
        const currentTag = this.getCurrentTag();
        if (currentTag) {
          this.addToken(currentTag, 'is', this.customValue.trim());
          this.customValue = '';
          this.closeAllDropdowns();
        }
      }
    },



    parseAndAddTokenInValuesMode(input) {
      if (input.includes(':')) {
  
        const parts = input.split(':', 2);
        if (parts.length === 2) {
          this.addToken(parts[0], 'is', parts[1]);
        } else {
          this.addRawTextToken(input);
        }
      } else {

        const currentTag = this.getCurrentTag();
        if (currentTag) {
          this.addToken(currentTag, 'is', input);
        } else {
          this.addRawTextToken(input);
        }
      }
    },

    clearAll() {
      this.tokens = [];
      this.inputValue = '';
    }
  };
}
