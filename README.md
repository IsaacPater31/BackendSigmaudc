# SIGMA-UDC — Backend

API REST del **Sistema Integral de Gestión de Matrícula Académica (SIGMA-UDC)**,
desarrollada para automatizar y optimizar el proceso de matrícula en la
Universidad de Cartagena.

## Descripción

El backend expone los servicios que soportan el proceso de matrícula académica:

- Autenticación institucional con tokens **JWT** y control de acceso por roles
  (`estudiante`, `jefe_departamental`).
- Gestión de **plazos** del periodo de matrícula (carga de documentos,
  inscripción de asignaturas y modificaciones).
- **Carga, consulta y validación de documentos** requeridos (certificado EPS y
  comprobante de pago).
- **Inscripción de asignaturas** con validación automática de prerrequisitos,
  créditos permitidos, cupos y cruce de horarios.
- **Solicitudes de modificación** de matrícula con aprobación de la jefatura y
  notificación en tiempo real (SSE).
- **Auditoría** de las operaciones críticas del sistema.
- Consulta de **pensum**, datos personales y horario del estudiante.

## Arquitectura

El proyecto sigue una arquitectura por capas con inyección de dependencias:

```
cmd/
└── main.go                # Punto de entrada: arranque, rutas y servidor HTTP
internal/
├── config/                # Carga y validación de variables de entorno
├── constants/             # Valores literales centralizados
├── database/              # Conexión PostgreSQL y migraciones
├── handlers/              # Controladores HTTP (orquestación de peticiones)
├── httpx/                 # Serialización uniforme de respuestas JSON
├── middleware/            # JWT, roles, CORS y registro de peticiones
├── models/                # Estructuras de datos y contratos de la API
├── repositories/          # Acceso a datos (SQL)
├── services/              # Lógica de negocio y reglas académicas
└── utils/                 # Utilidades (validación de contraseñas, red)
```

## Requisitos

- Go 1.24 o superior
- PostgreSQL 14 o superior

## Configuración

Copiar `.env.example` como `.env` y completar los valores:

| Variable        | Descripción                                        | Obligatoria |
|-----------------|----------------------------------------------------|-------------|
| `DATABASE_URL`  | Cadena de conexión a PostgreSQL                    | Sí          |
| `JWT_SECRET`    | Clave secreta para firmar los tokens JWT           | Sí          |
| `PORT`          | Puerto del servidor HTTP (por defecto `8080`)      | No          |
| `CORS_ORIGIN`   | Origen permitido para CORS (dominio del frontend)  | No          |
| `UPLOAD_DIR`    | Directorio de archivos subidos (`./uploads`)       | No          |

## Ejecución

```bash
go run ./cmd
```

El servidor escucha en `http://localhost:8080` (o el puerto configurado).

## Compilación y verificación

```bash
go build ./...     # compila todos los paquetes
go vet ./...       # análisis estático
gofmt -l .         # verificación de formato
go test ./...      # pruebas unitarias
```

## API

Rutas públicas: `POST /auth/login`, `POST /auth/set-password`.
El resto de endpoints vive bajo `/api` y requiere `Authorization: Bearer <JWT>`;
las rutas administrativas además exigen el rol `jefe_departamental`.
Los errores se responden siempre en JSON con el contrato `{"error": "..."}`.

## Autores y licencia

- Andrés Javier Quintana Alape
- Gustavo Calderón Kelsey
- Isaac David Paternina Noble

Universidad de Cartagena — Programa de Ingeniería de Sistemas, 2025.

Software propietario. Todos los derechos reservados (ver `LICENSE`).
