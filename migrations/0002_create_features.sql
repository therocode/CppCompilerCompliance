-- +goose Up
CREATE TABLE `features` (
  `name` TEXT,
  `timestamp` DATETIME,
  `cpp_version` INT NOT NULL,
  `paper_name` TEXT,
  `paper_link` TEXT,
  `gcc_support` BOOLEAN NOT NULL,
  `gcc_display_text` TEXT,
  `gcc_extra_text` TEXT,
  `clang_support` BOOLEAN NOT NULL,
  `clang_display_text` TEXT,
  `clang_extra_text` TEXT,
  `msvc_support` BOOLEAN NOT NULL,
  `msvc_display_text` TEXT,
  `msvc_extra_text` TEXT,
  `reported_to_twitter` BOOLEAN,
  `reported_broken` BOOLEAN,
  PRIMARY KEY (name, timestamp)
  );

-- +goose Down
DROP TABLE `features`;
