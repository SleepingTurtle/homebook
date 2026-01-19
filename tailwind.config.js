/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./web/templates/**/*.html"],
  theme: {
    extend: {
      colors: {
        primary: '#2563eb',
        'primary-hover': '#1d4ed8',
        success: '#16a34a',
        'success-bg': '#dcfce7',
        warning: '#ca8a04',
        'warning-bg': '#fef9c3',
        danger: '#dc2626',
        'danger-bg': '#fee2e2',
      }
    }
  },
  plugins: [],
}
