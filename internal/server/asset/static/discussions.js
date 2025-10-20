// Discussions Alpine.js functions

window.discussionFunctions = {
  filteredUsers: function() {
    return this.users.filter(user =>
      user.name.toLowerCase().includes(this.mentionFilter.toLowerCase())
    );
  },
  addMention: function(user) {
    const lastAtIndex = this.comment.lastIndexOf('@');
    if (lastAtIndex !== -1) {
      this.comment = this.comment.substring(0, lastAtIndex) + '@' + user.name + ' ';
    }
    this.mentionedUsers.push(user.id);
    this.showMentions = false;
    this.mentionFilter = '';
  },
  formatComment: function(comment) {
    let div = document.createElement('div');
    div.textContent = comment;
    let escaped = div.innerHTML;
    return escaped.replace(/@(\w+)/g, '<strong>@$1</strong>')
                  .replace(/\n/g, '<br>');
  },
  postComment: function() {
    htmx.ajax('POST', this.uri, {
      values: {
        content: this.comment,
        mentioned_users: this.mentionedUsers
      },
      swap: 'outerHTML',
      target: '#messages'
    });
    this.comment = '';
    this.mentionedUsers = [];
  }
};
