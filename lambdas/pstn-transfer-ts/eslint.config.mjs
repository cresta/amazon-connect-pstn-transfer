import eslint from "@eslint/js";
import prettierConfig from "eslint-config-prettier";
import prettier from "eslint-plugin-prettier";
import importPlugin from "eslint-plugin-import";
import tseslint from "typescript-eslint";

export default tseslint.config(
	eslint.configs.recommended,
	...tseslint.configs.recommendedTypeChecked,
	prettierConfig,
	{
		languageOptions: {
			parserOptions: {
				project: true,
				tsconfigRootDir: import.meta.dirname,
			},
		},
		plugins: {
			prettier: prettier,
			import: importPlugin,
		},
		rules: {
			"prettier/prettier": "error",
			"@typescript-eslint/no-unused-vars": [
				"error",
				{
					argsIgnorePattern: "^_",
					varsIgnorePattern: "^_",
				},
			],
			"@typescript-eslint/no-explicit-any": "warn",
			"@typescript-eslint/explicit-function-return-type": "off",
			"@typescript-eslint/explicit-module-boundary-types": "off",
			"@typescript-eslint/no-floating-promises": "error",
			"@typescript-eslint/no-misused-promises": [
				"error",
				{
					checksVoidReturn: false,
				},
			],
			// Enforce .js extensions in imports (required for ES modules)
			"import/extensions": [
				"error",
				"ignorePackages",
				{
					js: "always",
					ts: "never",
				},
			],
		},
	},
	{
		files: ["**/*.test.ts", "**/*.spec.ts"],
		extends: [tseslint.configs.disableTypeChecked],
		rules: {
			"@typescript-eslint/no-explicit-any": "off",
		},
	},
	{
		ignores: ["dist/**", "node_modules/**", "*.config.*"],
	},
);
