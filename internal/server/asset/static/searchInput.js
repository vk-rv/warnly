window.searchInput = function(initialTokens, filterCategories) {
  return {
    tokens: initialTokens || [],
    inputValue: '',
    showTagSuggestions: false,
    showOperatorDropdown: false,
    activeTokenIndex: -1,
    operatorDropdownPosition: { top: 0, left: 0 },

    filterCategories: filterCategories || [],
    
    init() {
      this.$watch('tokens', () => {
        window.dispatchEvent(new CustomEvent('tokens-changed', { detail: { tokens: this.tokens } }));
      });
    },
    
    focusInput() {
      this.$refs.searchInput.focus();
    },
    
    handleInputClick() {
      if (this.inputValue === '') {
        this.showTagSuggestions = true;
      }
    },
    
    handleInputFocus() {
      if (this.inputValue === '') {
        this.showTagSuggestions = true;
      }
    },
    
    handleInput() {
      if (this.inputValue === '') {
        this.showTagSuggestions = true;
      } else {
        this.showTagSuggestions = false;
      }
    },
    
    handleEnterKey() {
      if (this.inputValue.trim() !== '') {
        this.addRawTextToken(this.inputValue.trim());
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
      this.addToken(item.key, 'is', item.value);
    },
    
    setActiveCategory(index) {
      this.filterCategories.forEach((cat, i) => {
        cat.active = i === index;
      });
    },
    
    closeAllDropdowns() {
      this.showTagSuggestions = false;
      this.showOperatorDropdown = false;
    },
    
    clearAll() {
      this.tokens = [];
      this.inputValue = '';
    }
  };
}
