function queriesTable(queries) {
    const processedQueries = queries.map(q => ({
        id: q.normalized_query_hash,
        sql: q.normalized_query,
        avgDuration: q.avg_duration,
        avgDurationDisplay: q.avg_duration.toFixed(2),
        rowsScanned: q.avg_result_rows,
        rowsScannedDisplay: q.avg_result_rows.toFixed(2),
        callsPerMinute: q.calls_per_minute,
        callsPerMinuteDisplay: q.calls_per_minute.toFixed(2),
        progress: q.percent,
        totalReadBytes: q.total_read_bytes,
        totalReadBytesNumeric: q.total_read_bytes_numeric,
        percentageIOPS: q.percentage_iops,
        percentageIOPSDisplay: q.percentage_iops.toFixed(2),
        percentageRuntime: q.percentage_runtime,
        percentageRuntimeDisplay: q.percentage_runtime.toFixed(2),
        totalCalls: q.total_calls,
        totalCallsDisplay: q.total_calls.toString()
    }));
    return {
        queries: processedQueries,
        sortedQueries: [...processedQueries],
        activeTooltip: null,
        sortField: 'totalReadBytesNumeric',
        sortDirection: 'desc',

        init() {
            this.sortedQueries = [...this.queries];
            this.sortQueries();
        },

        copyToClipboard(text) {
            navigator.clipboard.writeText(text);
            showToast('Copied to clipboard');
            this.activeTooltip = null;
        },

        formatSql(sql) {
            const maxLength = 100;
            if (sql.length > maxLength) {
                sql = sql.substring(0, maxLength) + '...';
            }
            return sql.split(' ').map(word => {
                if (['SELECT', 'AS', 'FROM'].includes(word.toUpperCase())) {
                    return `<span class='text-black font-semibold'>${word}</span>`;
                }
                return word;
            }).join(' ');
        },

        toggleTooltip(id) {
            if (this.activeTooltip === id) {
                this.activeTooltip = null;
            } else {
                this.activeTooltip = id;
            }
        },

        sortBy(field) {
            if (this.sortField === field) {
                this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc';
            } else {
                this.sortField = field;
                this.sortDirection = 'asc';
            }
            this.sortQueries();
        },

        sortQueries() {
            this.sortedQueries = [...this.queries].sort((a, b) => {
                let aVal = a[this.sortField];
                let bVal = b[this.sortField];

                if (this.sortField === 'sql') {
                    aVal = aVal.toLowerCase();
                    bVal = bVal.toLowerCase();
                }

                if (aVal < bVal) {
                    return this.sortDirection === 'asc' ? -1 : 1;
                }
                if (aVal > bVal) {
                    return this.sortDirection === 'asc' ? 1 : -1;
                }
                return 0;
            });
        },

        getSortIcon(field) {
            if (this.sortField !== field) {
                return '<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4"></path></svg>';
            }
            if (this.sortDirection === 'asc') {
                return '<svg class="w-4 h-4 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7"></path></svg>';
            } else {
                return '<svg class="w-4 h-4 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>';
            }
        }
    };
}
