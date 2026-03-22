type User = {
  id: string;
  name: string;
  email: string;
  address?: string;
};

type UsersIndexProps = {
  users: User[];
  query: string;
  mode: "list" | "search";
  limit: number;
  offset: number;
  has_more: boolean;
};

export default function UsersIndex({ users = [], query, mode, limit, offset, has_more }: UsersIndexProps) {
  const prevOffset = Math.max(0, offset - limit);
  const nextOffset = offset + limit;

  return (
    <main style={{ fontFamily: "sans-serif", maxWidth: 960, margin: "0 auto", padding: "1rem" }}>
      <h1>Users</h1>

      <form method="get" action="/users" style={{ display: "flex", gap: "0.5rem", marginBottom: "1rem" }}>
        <input
          type="text"
          name="q"
          placeholder="Search by name or email"
          defaultValue={query}
          style={{ flex: 1, padding: "0.5rem" }}
        />
        <button type="submit">Search</button>
      </form>

      <p style={{ color: "#666" }}>
        Mode: <strong>{mode}</strong> • Results: <strong>{users.length}</strong>
      </p>

      <table width="100%" cellPadding={8} style={{ borderCollapse: "collapse" }}>
        <thead>
          <tr>
            <th align="left">ID</th>
            <th align="left">Name</th>
            <th align="left">Email</th>
          </tr>
        </thead>
        <tbody>
          {users.map((u) => (
            <tr key={u.id} style={{ borderTop: "1px solid #ddd" }}>
              <td>{u.id}</td>
              <td>{u.name}</td>
              <td>{u.email}</td>
            </tr>
          ))}
          {users.length === 0 ? (
            <tr>
              <td colSpan={3} style={{ color: "#666" }}>
                No users found.
              </td>
            </tr>
          ) : null}
        </tbody>
      </table>

      {mode === "list" ? (
        <nav style={{ marginTop: "1rem", display: "flex", gap: "0.5rem" }}>
          {offset > 0 ? (
            <a href={`/users?limit=${limit}&offset=${prevOffset}`}>Previous</a>
          ) : (
            <span style={{ color: "#ccc" }}>Previous</span>
          )}
          {has_more ? (
            <a href={`/users?limit=${limit}&offset=${nextOffset}`}>Next</a>
          ) : (
            <span style={{ color: "#ccc" }}>Next</span>
          )}
        </nav>
      ) : null}
    </main>
  );
}
