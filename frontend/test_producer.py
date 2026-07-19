import pika
import json
import uuid

RABBITMQ_URL = "amqp://guest:guest@localhost:5672/"
queue_name = "email.send"

def main():
    trace_id = str(uuid.uuid4())

    message = {
        "trace_id": trace_id,
        "tenant_id": 1,
        "service_id": 101,
        "receiver_email": "niti2002sh@gmail.com",
        "subject": "RabbitMQ Test Email With Image",
        "template": "welcome",
        "data": {
            "name": "Nitesh"
        },
        "attachments": [
            {
                "name": "sample-image.png",
                "url": "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTFYqoKTu_o3Zns2yExbst2Co84Gpc2Q1RJbA&s"
            }
        ]
    }

    connection = pika.BlockingConnection(
        pika.URLParameters(RABBITMQ_URL)
    )
    channel = connection.channel()

    channel.queue_declare(queue=queue_name, durable=True)

    channel.basic_publish(
        exchange="",
        routing_key=queue_name,
        body=json.dumps(message),
        properties=pika.BasicProperties(
            delivery_mode=2,
            headers={"x-trace-id": trace_id}
        ),
    )

    print("✅ Message sent")
    print("Trace ID:", trace_id)

    connection.close()


if __name__ == "__main__":
    main()