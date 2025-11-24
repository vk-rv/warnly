window.eventFilters = function(initialData) {
    return {
        projectId: initialData.projectId || '',
        issueId: initialData.issueId || '',
        contextMenu: {
            show: false,
            x: 0,
            y: 0,
            column: '',
            value: ''
        },
        hideContextMenu() {
            this.contextMenu.show = false;
        },
        events: initialData.events || [],
        navigateToEvent(eventId) {
            const url = new URL('/projects/' + this.projectId + '/issues/' + this.issueId, window.location.origin);
            url.searchParams.set('event_id', eventId);
            url.searchParams.set('period', this.period);
            url.searchParams.set('source', 'issue');
            htmx.ajax('GET', url.toString(), {
                target: '#issue_content',
                swap: 'outerHTML settle:0',
            });
        },
        offset: initialData.offset || 0,
        period: initialData.period,
        startDate: initialData.start || '',
        endDate: initialData.end || '',
        eventCount: initialData.eventCount || 0,
        totalErrors: initialData.totalErrors || 0,
        searchQuery: initialData.searchQuery || '',
        
        init() {
            window.addEventListener('period-changed', (e) => {
                this.period = e.detail.period;
                this.startDate = '';
                this.endDate = '';
                this.offset = 0;
                this.reloadEvents();
            });

            window.addEventListener('absolute-range-changed', (e) => {
                this.period = '';
                this.startDate = e.detail.start;
                this.endDate = e.detail.end;
                this.offset = 0;
                this.reloadEvents();
            });
        },

        handleTokensChanged(tokens) {
            const queryParts = [];
            tokens.forEach(token => {
                if (token.isRawText) {
                    let value = token.value;
                    if (value.includes(' ')) {
                        value = "\"" + value + "\"";
                    }
                    queryParts.push(value);
                } else {
                    let op = '';
                    if (token.operator === 'is not') {
                        op = '!';
                    }
                    let value = token.value;
                    if (value.includes(' ')) {
                        value = "\"" + value + "\"";
                    }
                    queryParts.push(token.key + ":" + op + value);
                }
            });
            const newQuery = queryParts.join(' ');
            if (this.searchQuery !== newQuery) {
                this.searchQuery = newQuery;
                this.offset = 0;
                this.reloadEvents();
            }
        },
        reloadEvents() {
            const url = new URL('/projects/' + this.projectId + '/issues/' + this.issueId + '/events', window.location.origin);
            url.searchParams.set('offset', this.offset);
            if (this.period) {
                url.searchParams.set('period', this.period);
            }
            if (this.startDate && this.endDate) {
                url.searchParams.set('start', this.startDate);
                url.searchParams.set('end', this.endDate);
            }
            url.searchParams.set('query', this.searchQuery);
            htmx.ajax('GET', url.toString(), {
                target: '#issue_content',
                swap: 'outerHTML settle:0',
            }).then(() => {
                this.updateEventCount();
            });
        },
        paginatePrev() {
            if (this.offset <= 0) {
                this.offset = 0;
                return;
            }
            this.offset = Math.max(0, this.offset - 50);
            this.reloadEvents();
        },
        paginateNext() {
            if (this.offset + 50 >= this.totalErrors) {
                return;
			}
            this.offset += 50;
            this.reloadEvents();
        },
        updateEventCount() {
            const tableBody = document.querySelector('#eventtable tbody');
            if (tableBody) {
                this.eventCount = tableBody.rows.length;
            }
        }
    };
}
