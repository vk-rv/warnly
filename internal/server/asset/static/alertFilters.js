window.alertFilters = function(initialData) {
  return {
    team: initialData.team || '',
    project: initialData.project || '',
    offset: initialData.offset || 0,
    limit: 50,
    totalAlerts: initialData.totalAlerts || 0,

    init() {
      this.$el.addEventListener('team-changed', (e) => {
        this.team = e.detail.team;
        this.resetFilters();
        this.applyFilters();
      });

      this.$el.addEventListener('project-changed', (e) => {
        this.project = e.detail.project;
        this.resetFilters();
        this.applyFilters();
      });
    },

    resetFilters() {
      this.offset = 0;
    },

    paginateNext() {
      if (this.offset + this.limit >= this.totalAlerts) {
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
      return this.offset + this.limit < this.totalAlerts;
    },

    canGoPrev() {
      return this.offset > 0;
    },

    buildQueryParams() {
      const params = new URLSearchParams();

      if (this.team) {
        params.set('team_name', this.team);
      }

      if (this.project) {
        params.set('project_name', this.project);
      }

      params.set('offset', this.offset);

      return params.toString();
    },

    applyFilters() {
      const params = this.buildQueryParams();
      const url = '/alerts?' + params + '&partial=body';
      htmx.ajax('GET', url, {
        target: '#alerts-body',
        swap: 'outerHTML'
      }).then(() => {
        const alertsBody = document.getElementById('alerts-body');
        if (alertsBody) {
          const newTotal = parseInt(alertsBody.dataset.total) || 0;
          this.totalAlerts = newTotal;
        }
        if (window.Alpine) {
          window.Alpine.initTree(document.getElementById('alerts-body'));
        }
      });
    },

    getPaginationSummary() {
      if (this.totalAlerts === 0) {
        return 'Showing 0-0 of 0 alerts';
      }

      const start = this.offset + 1;
      const end = Math.min(this.offset + this.limit, this.totalAlerts);
      const currentPage = Math.floor(this.offset / this.limit) + 1;
      const totalPages = Math.ceil(this.totalAlerts / this.limit);

      return `Showing ${start}-${end} of ${this.totalAlerts} alerts (page ${currentPage} of ${totalPages})`;
    }
  };
}
