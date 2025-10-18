window.issueFilters = function(initialData) {
  return {
    filters: initialData.filters || [],
    searchQuery: initialData.searchQuery || '',
    selectedPeriod: initialData.period || '14d',
    selectedProject: initialData.projectName || '',
    startDate: '',
    endDate: '',

    offset: initialData.offset || 0,
    limit: 50,
    totalIssues: initialData.totalIssues || 0,

    contextMenu: {
      show: false,
      x: 0,
      y: 0,
      column: '',
      value: ''
    },

    init() {
      window.addEventListener('period-changed', (e) => {
        this.selectedPeriod = e.detail.period;
        this.startDate = '';
        this.endDate = '';
        this.applyFilters();
      });

      window.addEventListener('absolute-range-changed', (e) => {
        this.selectedPeriod = '';
        this.startDate = e.detail.start;
        this.endDate = e.detail.end;
        this.applyFilters();
      });

      window.addEventListener('tokens-changed', (e) => {
        this.handleTokensChanged(e.detail.tokens);
      });

      this.$el.addEventListener('project-changed', (e) => {
        this.selectedProject = e.detail.project;
        this.resetFilters();
        this.applyFilters();
      });

      this.$el.addEventListener('issue-clicked', (e) => {
        this.navigateToIssue(e.detail.projectId, e.detail.issueId);
      });
    },

    handleTokensChanged(tokens) {
      const queryParts = [];
      tokens.forEach(token => {
        if (token.isRawText) {
          queryParts.push(token.value);
        } else {
          queryParts.push(`${token.key}${token.operator}${token.value}`);
        }
      });
      this.searchQuery = queryParts.join(' ');
      this.applyFilters();
    },

    addFilter(key, operator, value) {
      this.filters.push({ key, operator: operator || 'is', value });
      this.contextMenu.show = false;
    },
    
    removeFilter(index) {
      this.filters.splice(index, 1);
    },
    
    excludeFromFilter(column, value) {
      this.addFilter(column, 'is not', value);
    },
    
    resetFilters() {
      this.filters = [];
      this.searchQuery = '';
      this.offset = 0;
    },
    
    clearAll() {
      this.resetFilters();
      this.applyFilters();
    },

    showContextMenu(event, column, value) {
      event.preventDefault();
      this.contextMenu = {
        show: true,
        x: event.clientX,
        y: event.clientY,
        column,
        value
      };
    },
    
    hideContextMenu() {
      this.contextMenu.show = false;
    },

    paginateNext() {
      if (this.offset + this.limit >= this.totalIssues) {
        return;
      }
      this.offset += this.limit;
      this.applyFilters();
    },
    
    paginatePrev() {
      if (this.offset <= 0) {
        return;
      }
      this.offset = Math.max(0, this.offset - this.limit);
      this.applyFilters();
    },
    
    canGoNext() {
      return this.offset + this.limit < this.totalIssues;
    },
    
    canGoPrev() {
      return this.offset > 0;
    },

    buildQueryParams() {
      const params = new URLSearchParams();
      
      if (this.selectedProject) {
        params.set('project_name', this.selectedProject);
      }
      
      if (this.selectedPeriod) {
        params.set('period', this.selectedPeriod);
      }
      
      if (this.startDate && this.endDate) {
        params.set('start', this.startDate);
        params.set('end', this.endDate);
      }
      
      if (this.searchQuery) {
        params.set('query', this.searchQuery);
      }
      
      if (this.filters.length > 0) {
        params.set('filters', JSON.stringify(this.filters));
      }
      
      params.set('offset', this.offset);
      
      return params.toString();
    },

    applyFilters() {
      const params = this.buildQueryParams();
      const url = '/?' + params + '&partial=body';
      htmx.ajax('GET', url, {
        target: '#issues-body',
        swap: 'outerHTML'
      }).then(() => {
        if (window.Alpine) {
          window.Alpine.initTree(document.getElementById('issues-body'));
        }
      });
    },

    navigateToIssue(projectId, issueId) {
      const url = `/projects/${projectId}/issues/${issueId}`;
      htmx.ajax('GET', url, {
        target: '#content',
        swap: ' settle:0'
      });
    },

    getPaginationSummary() {
      if (this.totalIssues === 0) {
        return 'Showing 0-0 of 0 issues';
      }
      
      const start = this.offset + 1;
      const end = Math.min(this.offset + this.limit, this.totalIssues);
      const currentPage = Math.floor(this.offset / this.limit) + 1;
      const totalPages = Math.ceil(this.totalIssues / this.limit);
      
      return `Showing ${start}-${end} of ${this.totalIssues} issues (page ${currentPage} of ${totalPages})`;
    }
  };
}
