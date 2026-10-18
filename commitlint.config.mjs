export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // Preserve full explanations and references in the squash commit body.
    'body-max-line-length': [0],
    'footer-max-line-length': [0],
  },
};
