import { Activity, DollarSign, Zap, AlertTriangle } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

const stats = [
  {
    name: 'Total Traces',
    value: '0',
    icon: Activity,
    description: 'Last 24 hours',
  },
  {
    name: 'Total Cost',
    value: '$0.00',
    icon: DollarSign,
    description: 'Last 24 hours',
  },
  {
    name: 'Avg Latency',
    value: '0ms',
    icon: Zap,
    description: 'Last 24 hours',
  },
  {
    name: 'Error Rate',
    value: '0%',
    icon: AlertTriangle,
    description: 'Last 24 hours',
  },
];

export function DashboardPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground">
          Overview of your LLM application performance
        </p>
      </div>

      {/* Stats grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => (
          <Card key={stat.name}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">{stat.name}</CardTitle>
              <stat.icon className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stat.value}</div>
              <p className="text-xs text-muted-foreground">
                {stat.description}
              </p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Getting started */}
      <Card>
        <CardHeader>
          <CardTitle>Get Started</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-muted-foreground">
            Welcome to OTelGuard! To start collecting traces from your LLM
            application:
          </p>
          <ol className="list-decimal list-inside space-y-2 text-sm">
            <li>Install the OTelGuard SDK in your project</li>
            <li>Configure the SDK with your API key</li>
            <li>Instrument your LLM calls</li>
            <li>Start seeing traces in the dashboard</li>
          </ol>
          <div className="bg-muted rounded-lg p-4 font-mono text-sm">
            <p className="text-muted-foreground"># Python</p>
            <p>pip install otelguard-sdk</p>
            <br />
            <p className="text-muted-foreground"># JavaScript</p>
            <p>npm install @otelguard/sdk</p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
