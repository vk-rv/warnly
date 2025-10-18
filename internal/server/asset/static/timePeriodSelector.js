window.timePeriodSelector = function(initialPeriod) {
  return {
    isOpen: false,
    showCalendar: false,
    selectedPreset: initialPeriod || '14d',
    displayLabel: '',
    
    customRangeInput: '',
    customRangeError: '',
    
    customStartDate: null,
    customEndDate: null,
    
    currentMonth: new Date().getMonth(),
    currentYear: new Date().getFullYear(),
    monthNames: ['January', 'February', 'March', 'April', 'May', 'June', 
                 'July', 'August', 'September', 'October', 'November', 'December'],
    
    startDate: null,
    endDate: null,
    startHour: 12,
    startMinute: 0,
    startPeriod: 'AM',
    endHour: 12,
    endMinute: 0,
    endPeriod: 'PM',
    startSelectedPart: 'hour',
    endSelectedPart: 'hour',
    
    presets: [
      { label: 'Last hour', value: '1h' },
      { label: 'Last 24 hours', value: '24h' },
      { label: 'Last 7 days', value: '7d' },
      { label: 'Last 14 days', value: '14d' },
      { label: 'Last 30 days', value: '30d' },
      { label: 'Last 90 days', value: '90d' }
    ],
    
    init() {
      this.updateDisplayLabel();
    },
    
    toggleDropdown() {
      this.isOpen = !this.isOpen;
      if (!this.isOpen) {
        this.showCalendar = false;
      }
    },
    
    selectPreset(value) {
      this.selectedPreset = value;
      this.updateDisplayLabel();
      this.isOpen = false;
      this.showCalendar = false;
      
      window.dispatchEvent(new CustomEvent('period-changed', { detail: { period: value } }));
    },
    
    applyCustomRange() {
      const regex = /^(\d+)([smhdw])$/;
      const match = this.customRangeInput.match(regex);
      
      if (!match) {
        this.customRangeError = 'Invalid format. Use: 2h, 4d, 8w, etc.';
        return;
      }
      
      this.customRangeError = '';
      this.selectedPreset = this.customRangeInput;
      this.updateDisplayLabel();
      this.isOpen = false;
      
      window.dispatchEvent(new CustomEvent('period-changed', { detail: { period: this.customRangeInput } }));
    },
    
    openCalendar() {
      this.showCalendar = true;
    },
    
    closeCalendar() {
      this.showCalendar = false;
    },
    
    prevMonth() {
      if (this.currentMonth === 0) {
        this.currentMonth = 11;
        this.currentYear--;
      } else {
        this.currentMonth--;
      }
    },
    
    nextMonth() {
      if (this.currentMonth === 11) {
        this.currentMonth = 0;
        this.currentYear++;
      } else {
        this.currentMonth++;
      }
    },
    
    applyAbsoluteRange() {
      if (!this.startDate || !this.endDate) {
        return;
      }
      
      const start = this.formatDateTime(this.startDate, this.startHour, this.startMinute, this.startPeriod);
      const end = this.formatDateTime(this.endDate, this.endHour, this.endMinute, this.endPeriod);
      
      this.selectedPreset = 'custom-range';

      this.customStartDate = this.startDate;
      this.customEndDate = this.endDate;
      this.updateDisplayLabel();
      this.isOpen = false;
      this.showCalendar = false;
      
      window.dispatchEvent(new CustomEvent('absolute-range-changed', { detail: { start, end } }));
    },
    
    formatDateTime(dateStr, hour, minute, period) {
      let hour24 = hour;
      if (period === 'PM' && hour !== 12) {
        hour24 = hour + 12;
      } else if (period === 'AM' && hour === 12) {
        hour24 = 0;
      }
      
      let datePart;
      if (typeof dateStr === 'string') {
        datePart = dateStr; // Already in YYYY-MM-DD format
      } else if (dateStr instanceof Date) {
        const year = dateStr.getFullYear();
        const month = String(dateStr.getMonth() + 1).padStart(2, '0');
        const day = String(dateStr.getDate()).padStart(2, '0');
        datePart = `${year}-${month}-${day}`;
      } else {
        return '';
      }
      
      const hourStr = String(hour24).padStart(2, '0');
      const minuteStr = String(minute).padStart(2, '0');
      
      return `${datePart}T${hourStr}:${minuteStr}:00`;
    },
    
    updateDisplayLabel() {
      const preset = this.presets.find(p => p.value === this.selectedPreset);
      if (preset) {
        this.displayLabel = preset.label;
      } else if (this.selectedPreset === 'custom-range') {
        if (this.customStartDate && this.customEndDate) {
          const startFormatted = this.formatDateForDisplay(this.customStartDate);
          const endFormatted = this.formatDateForDisplay(this.customEndDate);
          this.displayLabel = `${startFormatted} - ${endFormatted}`;
        } else {
          this.displayLabel = 'Custom range';
        }
      } else {
        this.displayLabel = this.selectedPreset;
      }
    },
    
    formatDateForDisplay(dateStr) {
      if (!dateStr) return '';
      
      const date = new Date(dateStr + 'T00:00:00');
      const options = { month: 'short', day: 'numeric', year: 'numeric' };
      return date.toLocaleDateString('en-US', options);
    },
    
    getDaysInMonth() {
      return new Date(this.currentYear, this.currentMonth + 1, 0).getDate();
    },
    
    getFirstDayOfMonth() {
      return new Date(this.currentYear, this.currentMonth, 1).getDay();
    }
  };
}
