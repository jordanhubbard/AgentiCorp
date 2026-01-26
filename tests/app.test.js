const { sum, multiply } = require('../src/app');

test('sum function', () => {
  expect(sum(1, 2)).toBe(3);
});

test('multiply function', () => {
  expect(multiply(3, 4)).toBe(12);
});

module.exports = { sum, multiply };
