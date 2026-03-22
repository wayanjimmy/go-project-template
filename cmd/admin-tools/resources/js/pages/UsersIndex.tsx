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
    <main className="min-h-screen bg-slate-50 px-4 py-8 text-slate-900 sm:px-6 lg:px-8">
      <div className="mx-auto w-full max-w-5xl space-y-6">
        <header className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Users</h1>
          <p className="text-sm text-slate-600">Search and browse user records.</p>
        </header>

        <form method="get" action="/users" className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm sm:flex-row">
          <input
            type="text"
            name="q"
            placeholder="Search by name or email"
            defaultValue={query}
            className="w-full flex-1 rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none ring-indigo-200 transition focus:border-indigo-500 focus:ring-2"
          />
          <button
            type="submit"
            className="inline-flex items-center justify-center rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-500"
          >
            Search
          </button>
        </form>

        <div className="rounded-xl border border-slate-200 bg-white p-4 text-sm text-slate-700 shadow-sm">
          Mode: <strong className="font-semibold text-slate-900">{mode}</strong>
          <span className="mx-2 text-slate-300">•</span>
          Results: <strong className="font-semibold text-slate-900">{users.length}</strong>
        </div>

        <section className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200 text-sm">
              <thead className="bg-slate-50">
                <tr>
                  <th className="px-4 py-3 text-left font-medium text-slate-600">ID</th>
                  <th className="px-4 py-3 text-left font-medium text-slate-600">Name</th>
                  <th className="px-4 py-3 text-left font-medium text-slate-600">Email</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {users.map((u) => (
                  <tr key={u.id} className="hover:bg-slate-50">
                    <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-slate-600">{u.id}</td>
                    <td className="px-4 py-3 font-medium text-slate-900">{u.name}</td>
                    <td className="px-4 py-3 text-slate-700">{u.email}</td>
                  </tr>
                ))}
                {users.length === 0 ? (
                  <tr>
                    <td colSpan={3} className="px-4 py-8 text-center text-slate-500">
                      No users found.
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </section>

        {mode === "list" ? (
          <nav className="flex items-center gap-3">
            {offset > 0 ? (
              <a
                href={`/users?limit=${limit}&offset=${prevOffset}`}
                className="inline-flex items-center rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-50"
              >
                Previous
              </a>
            ) : (
              <span className="inline-flex items-center rounded-lg border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-400">
                Previous
              </span>
            )}
            {has_more ? (
              <a
                href={`/users?limit=${limit}&offset=${nextOffset}`}
                className="inline-flex items-center rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-50"
              >
                Next
              </a>
            ) : (
              <span className="inline-flex items-center rounded-lg border border-slate-200 bg-slate-100 px-3 py-2 text-sm text-slate-400">
                Next
              </span>
            )}
          </nav>
        ) : null}
      </div>
    </main>
  );
}
