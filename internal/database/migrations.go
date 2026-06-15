package database

import (
	"database/sql"
	"fmt"
)

// RunMigrations ensures that required tables for periodos y plazos exist
func RunMigrations(db *sql.DB) error {
	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS periodo_academico (
			id SERIAL PRIMARY KEY,
			year INT NOT NULL,
			semestre INT NOT NULL,
			activo BOOLEAN NOT NULL DEFAULT FALSE,
			archivado BOOLEAN NOT NULL DEFAULT FALSE,
			CONSTRAINT periodo_unico UNIQUE (year, semestre)
		)
		`,
		`
		CREATE TABLE IF NOT EXISTS plazos (
			id SERIAL PRIMARY KEY,
			periodo_id INT NOT NULL,
			programa_id INT NOT NULL,
			documentos BOOLEAN NOT NULL DEFAULT FALSE,
			inscripcion BOOLEAN NOT NULL DEFAULT FALSE,
			modificaciones BOOLEAN NOT NULL DEFAULT FALSE
		)
		`,
		`ALTER TABLE periodo_academico ADD COLUMN IF NOT EXISTS archivado BOOLEAN NOT NULL DEFAULT FALSE`,
		`UPDATE periodo_academico SET archivado = FALSE WHERE archivado IS NULL`,
		`UPDATE periodo_academico SET activo = FALSE WHERE archivado = TRUE`,
		`ALTER TABLE plazos ADD COLUMN IF NOT EXISTS programa_id INT`,
		`UPDATE plazos SET programa_id = 1 WHERE programa_id IS NULL`,
		`ALTER TABLE plazos ALTER COLUMN programa_id SET NOT NULL`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conname = 'fk_plazos_periodo' 
				AND conrelid = 'plazos'::regclass
			) THEN
				ALTER TABLE plazos
				ADD CONSTRAINT fk_plazos_periodo FOREIGN KEY (periodo_id) REFERENCES periodo_academico(id) ON DELETE CASCADE;
			END IF;
		END $$;
		`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conname = 'fk_plazos_programa' 
				AND conrelid = 'plazos'::regclass
			) THEN
				ALTER TABLE plazos
				ADD CONSTRAINT fk_plazos_programa FOREIGN KEY (programa_id) REFERENCES programa(id) ON DELETE CASCADE;
			END IF;
		END $$;
		`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conname = 'plazos_periodo_programa_unique' 
				AND conrelid = 'plazos'::regclass
			) THEN
				ALTER TABLE plazos
				ADD CONSTRAINT plazos_periodo_programa_unique UNIQUE (periodo_id, programa_id);
			END IF;
		END $$;
		`,
		`
		CREATE TABLE IF NOT EXISTS documentos_estudiante (
			id SERIAL PRIMARY KEY,
			estudiante_id INT NOT NULL,
			programa_id INT NOT NULL,
			periodo_id INT NOT NULL,
			tipo_documento VARCHAR(100) NOT NULL CHECK (tipo_documento IN ('certificado_eps', 'comprobante_matricula')),
			archivo_url TEXT NOT NULL,
			estado VARCHAR(20) DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'aprobado', 'rechazado')),
			observacion TEXT DEFAULT NULL,
			revisado_por INT DEFAULT NULL,
			fecha_subida TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			fecha_revision TIMESTAMP DEFAULT NULL
		)
		`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conname = 'fk_doc_estudiante' 
				AND conrelid = 'documentos_estudiante'::regclass
			) THEN
				ALTER TABLE documentos_estudiante
				ADD CONSTRAINT fk_doc_estudiante FOREIGN KEY (estudiante_id) REFERENCES estudiante(id) ON DELETE CASCADE;
			END IF;
		END $$;
		`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conname = 'fk_doc_programa' 
				AND conrelid = 'documentos_estudiante'::regclass
			) THEN
				ALTER TABLE documentos_estudiante
				ADD CONSTRAINT fk_doc_programa FOREIGN KEY (programa_id) REFERENCES programa(id);
			END IF;
		END $$;
		`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conname = 'fk_doc_periodo' 
				AND conrelid = 'documentos_estudiante'::regclass
			) THEN
				ALTER TABLE documentos_estudiante
				ADD CONSTRAINT fk_doc_periodo FOREIGN KEY (periodo_id) REFERENCES periodo_academico(id);
			END IF;
		END $$;
		`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conname = 'fk_doc_revisor' 
				AND conrelid = 'documentos_estudiante'::regclass
			) THEN
				ALTER TABLE documentos_estudiante
				ADD CONSTRAINT fk_doc_revisor FOREIGN KEY (revisado_por) REFERENCES jefe_departamental(id);
			END IF;
		END $$;
		`,
		`
		ALTER TABLE estudiante
		ADD COLUMN IF NOT EXISTS sexo VARCHAR(10) NOT NULL DEFAULT 'otro'
		`,
		`
		ALTER TABLE estudiante
		ADD COLUMN IF NOT EXISTS foto_perfil TEXT
		`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint
				WHERE conname = 'chk_estudiante_sexo'
				AND conrelid = 'estudiante'::regclass
			) THEN
				ALTER TABLE estudiante
				ADD CONSTRAINT chk_estudiante_sexo CHECK (sexo IN ('masculino','femenino','otro'));
			END IF;
		END $$;
		`,
		`
		ALTER TABLE jefe_departamental
		ADD COLUMN IF NOT EXISTS sexo VARCHAR(10) NOT NULL DEFAULT 'otro'
		`,
		`
		ALTER TABLE jefe_departamental
		ADD COLUMN IF NOT EXISTS foto_perfil TEXT
		`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint
				WHERE conname = 'chk_jefe_sexo'
				AND conrelid = 'jefe_departamental'::regclass
			) THEN
				ALTER TABLE jefe_departamental
				ADD CONSTRAINT chk_jefe_sexo CHECK (sexo IN ('masculino','femenino','otro'));
			END IF;
		END $$;
		`,
		// Tabla de solicitudes de modificación de matrícula
		`
		CREATE TABLE IF NOT EXISTS solicitud_modificacion (
			id SERIAL PRIMARY KEY,
			estudiante_id INT NOT NULL,
			programa_id INT NOT NULL,
			periodo_id INT NOT NULL,
			grupos_agregar JSONB DEFAULT '[]',
			grupos_retirar JSONB DEFAULT '[]',
			estado VARCHAR(20) DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'aprobada', 'rechazada')),
			observacion TEXT DEFAULT NULL,
			revisado_por INT DEFAULT NULL,
			fecha_solicitud TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			fecha_revision TIMESTAMP DEFAULT NULL
		)
		`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conname = 'fk_solicitud_estudiante' 
				AND conrelid = 'solicitud_modificacion'::regclass
			) THEN
				ALTER TABLE solicitud_modificacion
				ADD CONSTRAINT fk_solicitud_estudiante FOREIGN KEY (estudiante_id) REFERENCES estudiante(id) ON DELETE CASCADE;
			END IF;
		END $$;
		`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conname = 'fk_solicitud_programa' 
				AND conrelid = 'solicitud_modificacion'::regclass
			) THEN
				ALTER TABLE solicitud_modificacion
				ADD CONSTRAINT fk_solicitud_programa FOREIGN KEY (programa_id) REFERENCES programa(id);
			END IF;
		END $$;
		`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conname = 'fk_solicitud_periodo' 
				AND conrelid = 'solicitud_modificacion'::regclass
			) THEN
				ALTER TABLE solicitud_modificacion
				ADD CONSTRAINT fk_solicitud_periodo FOREIGN KEY (periodo_id) REFERENCES periodo_academico(id);
			END IF;
		END $$;
		`,
		`
		CREATE UNIQUE INDEX IF NOT EXISTS solicitud_modificacion_pendiente_unica_idx
		ON solicitud_modificacion (estudiante_id, periodo_id)
		WHERE estado = 'pendiente'
		`,
		`
		UPDATE grupo
		SET cupo_disponible = LEAST(GREATEST(cupo_disponible, 0), cupo_max)
		WHERE cupo_disponible < 0 OR cupo_disponible > cupo_max
		`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint
				WHERE conname = 'grupo_cupo_no_supera_max_check'
				AND conrelid = 'grupo'::regclass
			) THEN
				ALTER TABLE grupo
				ADD CONSTRAINT grupo_cupo_no_supera_max_check CHECK (cupo_disponible <= cupo_max);
			END IF;
		END $$;
		`,
		`
		ALTER TABLE horario_grupo
		ADD COLUMN IF NOT EXISTS componente VARCHAR(20) NOT NULL DEFAULT 'teoria'
		`,
		`
		UPDATE horario_grupo SET componente = 'teoria' WHERE componente IS NULL OR componente = ''
		`,
		`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint
				WHERE conname = 'horario_grupo_componente_check'
				AND conrelid = 'horario_grupo'::regclass
			) THEN
				ALTER TABLE horario_grupo
				ADD CONSTRAINT horario_grupo_componente_check
				CHECK (componente IN ('teoria', 'laboratorio'));
			END IF;
		END $$;
		`,
		`
		CREATE UNIQUE INDEX IF NOT EXISTS horario_grupo_grupo_lab_unique
		ON horario_grupo (grupo_id)
		WHERE componente = 'laboratorio'
		`,
		`
		INSERT INTO horario_grupo (grupo_id, dia, hora_inicio, hora_fin, salon, componente)
		SELECT v.grupo_id, v.dia, v.hora_inicio::time, v.hora_fin::time, v.salon, 'laboratorio'
		FROM (VALUES
			(15, 'JUEVES', '14:00:00', '16:00:00', 'LAB-FIS'),
			(16, 'MIERCOLES', '14:00:00', '16:00:00', 'LAB-FIS'),
			(20, 'MARTES', '14:00:00', '16:00:00', 'LAB-QUI'),
			(21, 'JUEVES', '14:00:00', '16:00:00', 'LAB-QUI'),
			(26, 'MIERCOLES', '14:00:00', '16:00:00', 'LAB-FIS2'),
			(27, 'JUEVES', '14:00:00', '16:00:00', 'LAB-FIS2'),
			(35, 'MIERCOLES', '14:00:00', '16:00:00', 'LAB-FIS3')
		) AS v(grupo_id, dia, hora_inicio, hora_fin, salon)
		WHERE EXISTS (SELECT 1 FROM grupo g WHERE g.id = v.grupo_id)
		  AND NOT EXISTS (
			SELECT 1 FROM horario_grupo hg
			WHERE hg.grupo_id = v.grupo_id AND hg.componente = 'laboratorio'
		  )
		`,
		`
		CREATE OR REPLACE FUNCTION public.obtener_semestre_por_creditos(est_id integer) RETURNS integer
		LANGUAGE plpgsql STABLE
		SET search_path = public
		AS $$
		DECLARE
		    v_pensum_id INT;
		    v_total_creditos_aprobados INT;
		    v_semestre_result INT;
		BEGIN
		    SELECT ep.pensum_id INTO v_pensum_id
		    FROM public.estudiante_pensum ep
		    WHERE ep.estudiante_id = est_id
		    LIMIT 1;

		    IF v_pensum_id IS NULL THEN
		        SELECT id INTO v_pensum_id FROM public.pensum WHERE activo = TRUE LIMIT 1;
		    END IF;

		    IF v_pensum_id IS NULL THEN
		        RAISE EXCEPTION 'No se encontró pensum para el estudiante % ni pensum activo.', est_id;
		    END IF;

		    SELECT COALESCE(SUM(a.creditos), 0) INTO v_total_creditos_aprobados
		    FROM public.historial_academico h
		    JOIN public.asignatura a ON a.id = h.id_asignatura
		    WHERE h.id_estudiante = est_id AND h.estado IN ('aprobada','convalidada');

		    SELECT cap.semestre INTO v_semestre_result
		    FROM public.creditos_acumulados_pensum cap
		    WHERE cap.pensum_id = v_pensum_id AND cap.creditos_acumulados > v_total_creditos_aprobados
		    ORDER BY cap.semestre LIMIT 1;

		    IF v_semestre_result IS NULL THEN
		        SELECT MAX(semestre) INTO v_semestre_result
		        FROM public.pensum_asignatura WHERE pensum_id = v_pensum_id;
		    END IF;

		    IF v_semestre_result < 1 THEN v_semestre_result := 1; END IF;
		    RETURN v_semestre_result;
		END;
		$$;
		`,
		`
		CREATE OR REPLACE FUNCTION public.trigger_actualizar_semestre_estudiante() RETURNS trigger
		LANGUAGE plpgsql
		SET search_path = public
		AS $$
		DECLARE
		    nuevo_semestre INT;
		    nuevo_promedio NUMERIC;
		BEGIN
		    nuevo_semestre := public.obtener_semestre_por_creditos(NEW.id_estudiante);

		    SELECT AVG(nota)
		    INTO nuevo_promedio
		    FROM public.historial_academico
		    WHERE id_estudiante = NEW.id_estudiante
		      AND nota IS NOT NULL
		      AND estado IN ('aprobada', 'reprobada');

		    UPDATE public.estudiante
		    SET semestre = nuevo_semestre,
		        promedio = nuevo_promedio
		    WHERE id = NEW.id_estudiante;

		    RETURN NEW;
		END;
		$$;
		`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("error running migrations: %w", err)
		}
	}

	return nil
}
