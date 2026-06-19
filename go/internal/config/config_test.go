package config

import "testing"

func TestRabbitMQURIVHostEscaping(t *testing.T) {
	base := Config{
		RabbitMQHost:     "rmq",
		RabbitMQPort:     "5672",
		RabbitMQUsername: "user",
		RabbitMQPassword: "pass",
	}

	cases := []struct {
		name  string
		vhost string
		want  string
	}{
		{name: "default slash", vhost: "/", want: "amqp://user:pass@rmq:5672/%2F"},
		{name: "empty defaults to slash", vhost: "", want: "amqp://user:pass@rmq:5672/%2F"},
		{name: "named vhost", vhost: "prod", want: "amqp://user:pass@rmq:5672/prod"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			c.RabbitMQVHost = tc.vhost
			got := c.RabbitMQURI()
			if got != tc.want {
				t.Fatalf("RabbitMQURI() = %q, want %q", got, tc.want)
			}
			if got == "amqp://user:pass@rmq:5672/%252F" {
				t.Fatalf("vhost double-escaped: %q", got)
			}
		})
	}
}
