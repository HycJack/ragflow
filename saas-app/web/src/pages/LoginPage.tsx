import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useLogin } from '@/hooks/queries';
import { setToken } from '@/api/client';
import { useWorkspace } from '@/store/workspace';
import { ErrorText } from '@/components/Ui';

export default function LoginPage() {
  const navigate = useNavigate();
  const setProfile = useWorkspace((s) => s.setProfile);
  const login = useLogin();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    login.mutate(
      { email, password },
      {
        onSuccess: (res) => {
          setToken(res.token);
          setProfile(res.profile);
          navigate('/onboarding');
        },
        onError: (err) => setError((err as Error).message),
      },
    );
  }

  return (
    <div className="auth-page">
      <div className="card" style={{ width: 380 }}>
        <div className="card-header">
          <h1>Welcome back</h1>
          <p>Sign in to your RAGFlow workspace</p>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="field">
            <label htmlFor="email">Email</label>
            <input
              id="email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
            />
          </div>
          <div className="field">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
            />
          </div>

          {error && <ErrorText>{error}</ErrorText>}

          <button className="btn btn-block" type="submit" disabled={login.isPending} style={{ marginTop: 8 }}>
            {login.isPending ? 'Signing in…' : 'Sign in'}
          </button>
        </form>

        <p className="muted" style={{ marginTop: 16, fontSize: 14, textAlign: 'center' }}>
          No account? <Link to="/register">Create one</Link>
        </p>
      </div>
    </div>
  );
}