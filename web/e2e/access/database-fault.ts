import { execFileSync } from "node:child_process";

export class AccountWriteFault {
  private readonly container: string;

  constructor() {
    const container = process.env.WANT_KEEP_ACCESS_DATABASE_CONTAINER;
    if (!container || !/^want-keep-access-\d+-\d+$/.test(container))
      throw new Error(
        "The access launcher must provide its disposable database",
      );
    const label = execFileSync(
      "docker",
      [
        "inspect",
        "--format",
        '{{index .Config.Labels "want-keep-test"}}',
        container,
      ],
      { encoding: "utf8" },
    ).trim();
    if (label !== "access") throw new Error("Refusing an unrelated database");
    this.container = container;
  }

  install() {
    this
      .execute(`CREATE FUNCTION want_keep.access_test_fail_write() RETURNS trigger LANGUAGE plpgsql AS $$
      BEGIN RAISE EXCEPTION 'synthetic account write failure' USING ERRCODE = 'XX000'; END $$;
      CREATE TRIGGER access_test_fail_write BEFORE INSERT ON want_keep.accounts
      FOR EACH ROW EXECUTE FUNCTION want_keep.access_test_fail_write();`);
  }

  remove() {
    this
      .execute(`DROP TRIGGER IF EXISTS access_test_fail_write ON want_keep.accounts;
      DROP FUNCTION IF EXISTS want_keep.access_test_fail_write();`);
  }

  private execute(sql: string) {
    execFileSync(
      "docker",
      [
        "exec",
        "-i",
        this.container,
        "psql",
        "-U",
        "postgres",
        "-d",
        "want_keep_test",
        "-v",
        "ON_ERROR_STOP=1",
      ],
      { input: sql, stdio: ["pipe", "pipe", "pipe"] },
    );
  }
}
