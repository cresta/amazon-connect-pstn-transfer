export default {
	preset: "ts-jest",
	testEnvironment: "node",
	extensionsToTreatAsEsm: [".ts"],
	roots: ["<rootDir>/src"],
	testMatch: ["**/*.test.ts"],
	testPathIgnorePatterns: ["/node_modules/", "/shared/"],
	collectCoverageFrom: ["src/**/*.ts", "!src/**/*.test.ts", "!src/**/*.d.ts"],
	moduleFileExtensions: ["ts", "js", "json"],
	transform: {
		"^.+\\.ts$": [
			"ts-jest",
			{
				useESM: true,
			},
		],
	},
	moduleNameMapper: {
		"^(\\.{1,2}/.*)\\.js$": "$1",
	},
};
